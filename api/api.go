package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ONSdigital/dp-graph/graph/driver"
	"github.com/ONSdigital/dp-hierarchy-api/models"
	"github.com/ONSdigital/go-ns/log"
	"github.com/gorilla/mux"
)

var hierarchyRoute *mux.Route

type API struct {
	store models.Storer
	host  string
	r     *mux.Router
}

func New(r *mux.Router, db models.Storer, url string) *API {
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

	log.InfoCtx(ctx, "attempting to get hierarchy root", logData)

	var err error
	var codelistID string
	if codelistID, err = api.store.GetHierarchyCodelist(ctx, instance, dimension); err != nil && err != driver.ErrNotFound {
		log.ErrorR(req, err, logData)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err == driver.ErrNotFound || codelistID == "" {
		log.DebugR(req, "hierarchy not found", logData)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var res *models.Response
	if res, err = api.store.GetHierarchyRoot(ctx, instance, dimension); err != nil {
		log.ErrorR(req, err, logData)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	res.AddLinks(api.host, instance, dimension, codelistID, true)

	b, err := json.Marshal(res)
	if err != nil {
		log.ErrorR(req, err, nil)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.InfoCtx(ctx, "get hierarchy root successful", logData)

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func (api *API) codesHandler(w http.ResponseWriter, req *http.Request) {
	instance := mux.Vars(req)["instance"]
	dimension := mux.Vars(req)["dimension"]
	code := mux.Vars(req)["code"]
	logData := log.Data{"instance_id": instance, "dimension": dimension, "code": code}
	ctx := req.Context()

	log.InfoCtx(ctx, "attempting to get hierarchy node for code", logData)

	var err error
	var codelistID string
	if codelistID, err = api.store.GetHierarchyCodelist(ctx, instance, dimension); err != nil && err != driver.ErrNotFound {
		log.ErrorR(req, err, logData)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err == driver.ErrNotFound || codelistID == "" {
		log.DebugR(req, "CodesHandler hierarchy not found", logData)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var res *models.Response
	if res, err = api.store.GetHierarchyElement(ctx, instance, dimension, code); err != nil && err != driver.ErrNotFound {
		log.ErrorR(req, err, logData)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err == driver.ErrNotFound || res.Label == "" {
		err = errors.New("incorrect code")
		log.ErrorR(req, err, logData)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	res.AddLinks(api.host, instance, dimension, codelistID, false)

	b, err := json.Marshal(res)
	if err != nil {
		log.ErrorR(req, err, nil)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.InfoCtx(ctx, "get hierarchy node for code successful", logData)

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}
