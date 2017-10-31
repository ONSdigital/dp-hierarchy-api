package api

import (
	"encoding/json"
	"net/http"

	"github.com/ONSdigital/dp-hierarchy-api/models"
	"github.com/ONSdigital/dp-hierarchy-api/store"
	"github.com/ONSdigital/go-ns/log"
	"github.com/gorilla/mux"
)

var (
	database        store.Storer
	hierarchyRoute  *mux.Route
	HierarchyAPIURL string
)

func SetDatabase(db store.Storer) {
	database = db
}

func AddRoutes(r *mux.Router) {
	r.Path("/hierarchies/{instance}/{dimension}").HandlerFunc(HierarchiesHandler).Name("hierarchy_url")
	r.Path("/hierarchies/{instance}/{dimension}/{code}").HandlerFunc(CodesHandler)
	hierarchyRoute = r.Get("hierarchy_url")
}

func HierarchiesHandler(w http.ResponseWriter, req *http.Request) {
	instance := mux.Vars(req)["instance"]
	dimension := mux.Vars(req)["dimension"]
	logData := log.Data{"instance_id": instance, "dimension": dimension}

	hierarchy := models.Hierarchy{URL: HierarchyAPIURL + req.URL.String(), InstanceId: instance, Dimension: dimension}

	var err error
	if hierarchy.CodelistId, err = database.GetCodelist(&hierarchy); err != nil {
		log.ErrorR(req, err, logData)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if hierarchy.CodelistId == "" {
		log.DebugR(req, "hierarchy not found", logData)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var res *models.Response
	if res, err = database.GetHierarchy(&hierarchy); err != nil {
		log.ErrorR(req, err, logData)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	res.AddLinks(&hierarchy, true)

	b, err := json.Marshal(res)
	if err != nil {
		log.ErrorR(req, err, nil)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func CodesHandler(w http.ResponseWriter, req *http.Request) {
	instance := mux.Vars(req)["instance"]
	dimension := mux.Vars(req)["dimension"]
	code := mux.Vars(req)["code"]
	logData := log.Data{"instance_id": instance, "dimension": dimension, "code": code}

	// get the full URL for the hierarchy (above .../code)
	hierPath, err := hierarchyRoute.URL("instance", instance, "dimension", dimension)
	if err != nil {
		log.ErrorR(req, err, logData)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	hierarchy := models.Hierarchy{URL: HierarchyAPIURL + hierPath.String(), InstanceId: instance, Dimension: dimension}

	if hierarchy.CodelistId, err = database.GetCodelist(&hierarchy); err != nil {
		log.ErrorR(req, err, logData)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if hierarchy.CodelistId == "" {
		log.DebugR(req, "CodesHandler hierarchy not found", logData)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var res *models.Response
	if res, err = database.GetCode(&hierarchy, code); err != nil {
		log.ErrorR(req, err, logData)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	res.AddLinks(&hierarchy, false)

	b, err := json.Marshal(res)
	if err != nil {
		log.ErrorR(req, err, nil)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}
