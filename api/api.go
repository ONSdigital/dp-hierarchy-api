package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ONSdigital/dp-graph/v2/graph/driver"
	dbmodels "github.com/ONSdigital/dp-graph/v2/models"
	"github.com/ONSdigital/dp-hierarchy-api/datastore"

	"github.com/ONSdigital/dp-hierarchy-api/models"
	"github.com/ONSdigital/log.go/v2/log"
	"github.com/gorilla/mux"
)

var hierarchyRoute *mux.Route

type API struct {
	store datastore.Storer
	host  string
	r     *mux.Router
}

func New(r *mux.Router, db datastore.Storer, url string) *API {
	api := &API{
		store: db,
		host:  url,
		r:     r,
	}

	api.r.Path("/hierarchies/{instance}/{dimension}").HandlerFunc(api.hierarchiesHandler).Name("hierarchy_url")
	api.r.Path("/hierarchies/{instance}/{dimension}/{code}").HandlerFunc(api.codesHandler)
	hierarchyRoute = api.r.Get("hierarchy_url")

	return api
}

func (api *API) hierarchiesHandler(w http.ResponseWriter, req *http.Request) {
	instance := mux.Vars(req)["instance"]
	dimension := mux.Vars(req)["dimension"]
	logData := log.Data{"instance_id": instance, "dimension": dimension}
	ctx := req.Context()

	log.Info(ctx, "attempting to get hierarchy root", logData)

	var err error
	var codelistID string
	if codelistID, err = api.store.GetHierarchyCodelist(ctx, instance, dimension); err != nil && err != driver.ErrNotFound {
		log.Error(ctx, "error getting hierarchy code list", err, logData)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err == driver.ErrNotFound || codelistID == "" {
		log.Error(ctx, "hierarchy not found", err, logData)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var dbRes *dbmodels.HierarchyResponse
	if dbRes, err = api.store.GetHierarchyRoot(ctx, instance, dimension); err != nil {
		log.Error(ctx, "error getting hierarchy root", err, logData)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	res := mapHierarchyResponse(dbRes)
	res.AddLinks(api.host, instance, dimension, codelistID, true)

	b, err := json.Marshal(res)
	if err != nil {
		log.Error(ctx, "error marshalling json response", err, logData)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Info(ctx, "get hierarchy root successful", logData)

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func (api *API) codesHandler(w http.ResponseWriter, req *http.Request) {
	instance := mux.Vars(req)["instance"]
	dimension := mux.Vars(req)["dimension"]
	code := mux.Vars(req)["code"]
	logData := log.Data{"instance_id": instance, "dimension": dimension, "code": code}
	ctx := req.Context()

	log.Info(ctx, "attempting to get hierarchy node for code", logData)

	var err error
	var codelistID string
	if codelistID, err = api.store.GetHierarchyCodelist(ctx, instance, dimension); err != nil && err != driver.ErrNotFound {
		log.Error(ctx, "error getting hierarchy code list", err, logData)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err == driver.ErrNotFound || codelistID == "" {
		log.Error(ctx, "hierarchy not found", err, logData)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var dbRes *dbmodels.HierarchyResponse
	if dbRes, err = api.store.GetHierarchyElement(ctx, instance, dimension, code); err != nil && err != driver.ErrNotFound {
		log.Error(ctx, "error getting hierarchy element", err, logData)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err == driver.ErrNotFound || dbRes.Label == "" {
		err = errors.New("incorrect code")
		log.Error(ctx, "code not found", err, logData)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	res := mapHierarchyResponse(dbRes)
	res.AddLinks(api.host, instance, dimension, codelistID, false)

	b, err := json.Marshal(res)
	if err != nil {
		log.Error(ctx, "error marshalling json response", err, logData)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Info(ctx, "get hierarchy node for code successful", logData)

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func mapHierarchyResponse(dbResponse *dbmodels.HierarchyResponse) models.Response {

	response := models.Response{
		ID:           dbResponse.ID,
		Label:        dbResponse.Label,
		Children:     mapHierarchyElements(dbResponse.Children),
		NoOfChildren: dbResponse.NoOfChildren,
		HasData:      dbResponse.HasData,
		Breadcrumbs:  mapHierarchyElements(dbResponse.Breadcrumbs),
		Order:        dbResponse.Order,
	}

	return response
}

func mapHierarchyElements(dbElements []*dbmodels.HierarchyElement) []*models.Element {

	var elements []*models.Element

	for _, dbElement := range dbElements {

		element := &models.Element{
			ID:           dbElement.ID,
			Label:        dbElement.Label,
			NoOfChildren: dbElement.NoOfChildren,
			HasData:      dbElement.HasData,
			Order:        dbElement.Order,
		}

		elements = append(elements, element)
	}

	return elements
}
