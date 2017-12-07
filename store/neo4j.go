package store

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/ONSdigital/dp-hierarchy-api/models"
	"github.com/ONSdigital/go-ns/log"
	bolt "github.com/ONSdigital/golang-neo4j-bolt-driver"
	"github.com/ONSdigital/golang-neo4j-bolt-driver/structures/graph"
)

const (
	pingStmt        = "MATCH (i) RETURN i LIMIT 1"
	existStmt       = "MATCH (i:`_hierarchy_node_%s_%s`) RETURN i LIMIT 1"
	getHierStmt     = "MATCH (i:`_hierarchy_node_%s_%s`) WHERE NOT (i)-[:hasParent]->() RETURN i LIMIT 1" // TODO check if this LIMIT is valid
	getCodeStmt     = "MATCH (i:`_hierarchy_node_%s_%s` {code:{code}}) RETURN i"
	getChildrenStmt = "MATCH (i:`_hierarchy_node_%s_%s` {code:{code}})<-[r:hasParent]-(child) RETURN child ORDER BY child.label"
	getAncestryStmt = "MATCH (i:`_hierarchy_node_%s_%s` {code:{code}})-[r:hasParent *]->(parent) RETURN parent"
)

// Store contains DB details
type Store struct {
	dbPool   bolt.ClosableDriverPool
	lastPing time.Time
}

type neoArgMap map[string]interface{}

// New creates a new Storer object
func New(dbURL string) (models.Storer, error) {
	pool, err := bolt.NewClosableDriverPool(dbURL, 5)
	if err != nil {
		log.Error(err, nil)
		return nil, err
	}
	return &Store{dbPool: pool, lastPing: time.Now()}, nil
}

// Close allows main to close the database connections with context
func (s *Store) Close(ctx context.Context) error {
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

func (s *Store) Ping(ctx context.Context) error {
	if time.Since(s.lastPing) < 1*time.Second {
		return nil
	}

	s.lastPing = time.Now()
	pingDoneChan := make(chan error)
	go func() {
		log.Trace("db ping", nil)
		if _, err := s.getProps(pingStmt); err != nil {
			log.ErrorC("Ping getAll", err, nil)
			pingDoneChan <- err
			return
		}
		close(pingDoneChan)
	}()
	select {
	case err := <-pingDoneChan:
		return err
	case <-ctx.Done():
		close(pingDoneChan)
		return ctx.Err()
	}
}

// GetCodelist obtains the codelist id for this hierarchy (also, check that it exists)
func (s *Store) GetCodelist(hierarchy *models.Hierarchy) (string, error) {
	neoStmt := fmt.Sprintf(existStmt, hierarchy.InstanceId, hierarchy.Dimension)
	props, err := s.getProps(neoStmt)
	if err != nil {
		log.ErrorC("GetCodelist getProps", err, nil)
		return "", err
	}
	if props == nil {
		// no results
		return "", nil
	}
	return props["code_list"].(string), nil
}

func (s *Store) getProps(neoStmt string) (res map[string]interface{}, err error) {
	logData := log.Data{"statement": neoStmt}
	conn, err := s.dbPool.OpenPool()
	if err != nil {
		log.ErrorC("getProps OpenPool", err, logData)
		return nil, err
	}
	defer conn.Close()

	rows, err := conn.QueryNeo(neoStmt, nil)
	if err != nil {
		log.ErrorC("getProps query", err, logData)
		return nil, err
	}
	defer rows.Close()

	data, _, err := rows.All()
	if err != nil {
		log.ErrorC("getProps rows.All", err, logData)
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}
	return data[0][0].(graph.Node).Properties, nil
}

// GetHierarchy returns the upper-most node for a given hierarchy
func (s *Store) GetHierarchy(hierarchy *models.Hierarchy) (*models.Response, error) {
	neoStmt := fmt.Sprintf(getHierStmt, hierarchy.InstanceId, hierarchy.Dimension)
	return s.queryResponse(hierarchy, neoStmt, neoArgMap{})
}

// GetCode gets a node in a given hierarchy for a given code
func (s *Store) GetCode(hierarchy *models.Hierarchy, code string) (res *models.Response, err error) {
	// res = &models.Response{}
	neoStmt := fmt.Sprintf(getCodeStmt, hierarchy.InstanceId, hierarchy.Dimension)
	if res, err = s.queryResponse(hierarchy, neoStmt, neoArgMap{"code": code}); err != nil {
		return
	}
	res.Breadcrumbs, err = s.getAncestry(hierarchy, code)
	return
}

// QueryResponse performs DB query (neoStmt, neoArgs) returning Response (should be singular)
func (s *Store) queryResponse(hierarchy *models.Hierarchy, neoStmt string, neoArgs neoArgMap) (res *models.Response, err error) {
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
	var row []interface{}
	var meta map[string]interface{}
	for row, meta, err = rows.NextNeo(); err == nil; row, meta, err = rows.NextNeo() {
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
		res.NoOfChildren = props["numberOfChildren"].(int64)
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
	res.Children, err = s.getChildren(hierarchy, res.ID)
	return
}

func (s *Store) getChildren(hierarchy *models.Hierarchy, code string) ([]*models.Element, error) {
	neoStmt := fmt.Sprintf(getChildrenStmt, hierarchy.InstanceId, hierarchy.Dimension)
	return s.queryElements(neoStmt, neoArgMap{"code": code}, hierarchy)
}

// GetAncestry retrieves a list of ancestors for this code - as breadcrumbs (ordered, nearest first)
func (s *Store) getAncestry(hierarchy *models.Hierarchy, code string) (ancestors []*models.Element, err error) {
	logData := log.Data{"instance_id": hierarchy.InstanceId, "dimension": hierarchy.Dimension, "code": code}
	neoStmt := fmt.Sprintf(getAncestryStmt, hierarchy.InstanceId, hierarchy.Dimension)
	if ancestors, err = s.queryElements(neoStmt, neoArgMap{"code": code}, hierarchy); err != nil {
		log.ErrorC("GetAncestry query", err, logData)
	} else if generationCount := len(ancestors); generationCount > 0 {
		// fix top-most URL (root of hierarchy) to use (non-code) hierarchy URL
		ancestors[generationCount-1].Links["self"] = models.Link{ID: ancestors[generationCount-1].Links["self"].ID, HRef: hierarchy.URL}
	}
	return
}

// queryElements returns a list of models.Elements from the database
func (s *Store) queryElements(neoStmt string, neoArgs neoArgMap, hierarchy *models.Hierarchy) ([]*models.Element, error) {
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
