package store

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ONSdigital/dp-hierarchy-api/models"
	"github.com/ONSdigital/go-ns/log"
	bolt "github.com/johnnadratowski/golang-neo4j-bolt-driver"
	"github.com/johnnadratowski/golang-neo4j-bolt-driver/structures/graph"
)

const (
	existStmt       = "MATCH (i:`_hierarchy_node_%s_%s`) RETURN i LIMIT 1"
	getHierStmt     = "MATCH (i:`_hierarchy_node_%s_%s`) WHERE NOT (i)-[:hasParent]->() RETURN i LIMIT 1" // TODO check if this LIMIT is valid
	getCodeStmt     = "MATCH (i:`_hierarchy_node_%s_%s` {code:{code}}) RETURN i"
	getChildrenStmt = "MATCH (i:`_hierarchy_node_%s_%s` {code:{code}})<-[r:hasParent]-(child) RETURN child"
	getParentStmt   = "MATCH (i:`_hierarchy_node_%s_%s` {code:{code}})-[r:hasParent]->(parent) RETURN parent"
	getAncestryStmt = "MATCH (i:`_hierarchy_node_%s_%s` {code:{code}})-[r:hasParent *]->(parent) RETURN parent"
)

type Store struct {
	dbPool bolt.ClosableDriverPool
}

type neoArgMap map[string]interface{}

type Storer interface {
	Close(ctx context.Context) error
	GetCodelist(hierarchy *models.Hierarchy) (string, error)
	GetHierarchy(hierarchy *models.Hierarchy) (*models.Response, error)
	GetCode(hierarchy *models.Hierarchy, code string) (*models.Response, error)
}

func New(dbUrl string) (Storer, error) {
	pool, err := bolt.NewClosableDriverPool(dbUrl, 5)
	if err != nil {
		log.Error(err, nil)
		return nil, err
	}
	return &Store{dbPool: pool}, nil
}

func (s Store) Close(ctx context.Context) error {
	errChan := make(chan error)
	go func() {
		errChan <- s.dbPool.Close()
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s Store) GetCodelist(hierarchy *models.Hierarchy) (string, error) {
	neoStmt := fmt.Sprintf(existStmt, hierarchy.InstanceId, hierarchy.Dimension)
	logData := log.Data{"statement": neoStmt}

	log.Trace("executing exists query", logData)
	conn, err := s.dbPool.OpenPool()
	if err != nil {
		return "", err
	}
	defer conn.Close()
	rows, err := conn.QueryNeo(neoStmt, nil)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	data, _, err := rows.All()
	if len(data) == 0 {
		return "", nil
	}
	props := data[0][0].(graph.Node).Properties
	return props["codeList"].(string), nil
}

func (s Store) GetHierarchy(hierarchy *models.Hierarchy) (*models.Response, error) {
	neoStmt := fmt.Sprintf(getHierStmt, hierarchy.InstanceId, hierarchy.Dimension)
	return s.QueryResponse(hierarchy, neoStmt, neoArgMap{})
}

func (s Store) GetCode(hierarchy *models.Hierarchy, code string) (res *models.Response, err error) {
	// res = &models.Response{}
	neoStmt := fmt.Sprintf(getCodeStmt, hierarchy.InstanceId, hierarchy.Dimension)
	if res, err = s.QueryResponse(hierarchy, neoStmt, neoArgMap{"code": code}); err != nil {
		return
	}
	res.Breadcrumbs, err = s.GetAncestry(hierarchy, code)
	return
}

// QueryResponse performs DB query (neoStmt, neoArgs) returning Response (should be singular)
func (s Store) QueryResponse(hierarchy *models.Hierarchy, neoStmt string, neoArgs neoArgMap) (res *models.Response, err error) {
	logData := log.Data{"statement": neoStmt, "row_count": 0, "neo_args": neoArgs}

	log.Trace("QueryResponse executing get query", logData)
	conn, err := s.dbPool.OpenPool()
	if err != nil {
		return
	}
	defer conn.Close()
	rows, err := conn.QueryNeo(neoStmt, neoArgs)
	if err != nil {
		return
	}

	res = &models.Response{}
	countRows := 0
	for row, meta, err := rows.NextNeo(); err == nil; row, meta, err = rows.NextNeo() {
		countRows++
		logData["row_count"] = countRows
		logData["row"] = row
		logData["meta"] = meta
		log.Trace("QueryResponse db row", logData)
		if countRows > 1 {
			err = errors.New("QueryResponse: got more than one row from DB")
			break
		}
		props := row[0].(graph.Node).Properties
		res.ID = props["code"].(string)
		res.Label = props["label"].(string)
		res.HasData = props["hasData"].(bool)
		delete(logData, "row")
		delete(logData, "meta")
	}
	rows.Close()
	if err == nil && countRows != 1 {
		err = errors.New("QueryResponse: got no rows from DB")
	}
	if err != nil && err != io.EOF {
		log.ErrorC("QueryResponse", err, logData)
		return
	}
	res.Children, err = s.GetChildren(hierarchy, res.ID)
	return
}

func (s Store) GetChildren(hierarchy *models.Hierarchy, code string) ([]*models.Element, error) {
	neoStmt := fmt.Sprintf(getChildrenStmt, hierarchy.InstanceId, hierarchy.Dimension)
	return s.QueryElements(neoStmt, neoArgMap{"code": code}, hierarchy)
}

// GetAncestry retrieves a list of ancestors for this code - as breadcrumbs (ordered, nearest first)
func (s Store) GetAncestry(hierarchy *models.Hierarchy, code string) (ancestors []*models.Element, err error) {
	logData := log.Data{"instance_id": hierarchy.InstanceId, "dimension": hierarchy.Dimension, "code": code}
	neoStmt := fmt.Sprintf(getAncestryStmt, hierarchy.InstanceId, hierarchy.Dimension)
	if ancestors, err = s.QueryElements(neoStmt, neoArgMap{"code": code}, hierarchy); err != nil {
		log.ErrorC("GetAncestry query", err, logData)
	} else if generation_count := len(ancestors); generation_count > 0 {
		// fix top-most URL (root of hierarchy) to use (non-code) hierarchy URL
		ancestors[generation_count-1].Links["self"] = models.Link{ID: ancestors[generation_count-1].Links["self"].ID, HRef: hierarchy.URL}
	}
	return
}

// QueryElements returns a list of models.Elements from the database
func (s Store) QueryElements(neoStmt string, neoArgs neoArgMap, hierarchy *models.Hierarchy) ([]*models.Element, error) {
	logData := log.Data{"db_statement": neoStmt, "row_count": 0, "db_args": neoArgs}
	log.Trace("QueryElements: executing get query", logData)
	conn, err := s.dbPool.OpenPool()
	if err != nil {
		log.ErrorC("QueryElements pool", err, logData)
		return nil, err
	}
	defer conn.Close()
	rows, err := conn.QueryNeo(neoStmt, neoArgs)
	if err != nil {
		log.ErrorC("QueryElements query", err, logData)
		if closeErr := rows.Close(); closeErr != nil {
			log.ErrorC("QueryElements close", closeErr, logData)
		}
		return nil, err
	}

	var (
		res       []*models.Element
		countRows = 0
	)
	for row, meta, err := rows.NextNeo(); err == nil; row, meta, err = rows.NextNeo() {
		countRows++
		logData["row"] = row
		logData["meta"] = meta
		logData["row_count"] = countRows
		log.Trace("QueryElements db row", logData)
		props := row[0].(graph.Node).Properties
		element := &models.Element{
			ID:           props["code"].(string),
			Label:        props["label"].(string),
			HasData:      props["hasData"].(bool),
			NoOfChildren: props["numberOfChildren"].(int64),
		}
		element.AddLinks(hierarchy)
		res = append(res, element)
		delete(logData, "row")
		delete(logData, "meta")
	}
	if closeErr := rows.Close(); closeErr != nil {
		log.ErrorC("QueryElements close ", closeErr, logData)
	}
	if err != nil && err != io.EOF {
		log.ErrorC("QueryElements", err, logData)
		return nil, err
	}

	return res, nil
}