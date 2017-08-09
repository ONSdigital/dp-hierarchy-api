package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"github.com/ONSdigital/dp-hierarchy-api/config"
	"github.com/ONSdigital/dp-hierarchy-api/response"
	"github.com/ONSdigital/dp-hierarchy-api/stubs"
	"github.com/ONSdigital/dp-hierarchy-api/validate"
	"github.com/ONSdigital/go-ns/log"
	"github.com/ONSdigital/go-ns/server"
	"github.com/gorilla/mux"
)

var hierarchy map[string]*stubs.Output

func main() {
	log.Namespace = "dp-hierarchy-api"
	config, err := config.Get()
	if err != nil {
		log.Error(err, nil)
		os.Exit(1)
	}

	hierarchy = stubs.GenerateHierarchy()

	router := mux.NewRouter()
	//router.Path("/hierarchies").HandlerFunc(codeListsHandler)
	router.Path("/hierarchies/{id}").HandlerFunc(hierarchiesHandler)
	router.Path("/hierarchies/{id}/{level1}").HandlerFunc(level1Handler)
	router.Path("/hierarchies/{id}/{level1}/{level2}").HandlerFunc(level2Handler)

	log.Debug("starting http server", log.Data{"bind_addr": config.BindAddr})
	srv := server.New(config.BindAddr, router)
	if err := srv.ListenAndServe(); err != nil {
		log.Error(err, nil)
		os.Exit(1)
	}
}

func hierarchiesHandler(w http.ResponseWriter, req *http.Request) {
	id := mux.Vars(req)["id"]
	if id != "CPI" {
		log.DebugR(req, "hierarchy not found", log.Data{"id": id})
		w.WriteHeader(http.StatusNotFound)
		return
	}

	res := &response.Response{
		ID:        hierarchy[id].ID,
		LabelCode: hierarchy[id].LabelCode,
		Label:     hierarchy[id].Label,
	}

	res.AddChildren(hierarchy[id].Children)
	res.AddLinks(req.URL.String())

	b, err := json.Marshal(res)
	if err != nil {
		log.ErrorR(req, err, nil)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func level1Handler(w http.ResponseWriter, req *http.Request) {
	id := mux.Vars(req)["id"]
	level1 := mux.Vars(req)["level1"]
	if id != "CPI" {
		log.DebugR(req, "hierarchy not found", log.Data{"id": id})
		w.WriteHeader(http.StatusNotFound)
		return
	}

	v := &validate.Request{
		R:  req,
		W:  w,
		ID: id,
	}

	if ok := v.Validate(level1); !ok {
		return
	}

	item := hierarchy[level1]
	res := &response.Response{
		ID:        item.ID,
		LabelCode: item.LabelCode,
		Label:     item.Label,
	}

	res.AddChildren(item.Children)
	if data := res.AddParent(item.Parents, level1); data != nil {
		log.ErrorR(req, errors.New("too many parent elements found"), *data)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	res.AddLinks(req.URL.String())

	b, err := json.Marshal(res)
	if err != nil {
		log.ErrorR(req, err, nil)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func level2Handler(w http.ResponseWriter, req *http.Request) {
	id := mux.Vars(req)["id"]
	level1 := mux.Vars(req)["level1"]
	level2 := mux.Vars(req)["level2"]
	label := level1 + "." + level2
	if id != "CPI" {
		log.DebugR(req, "hierarchy not found", log.Data{"id": id})
		w.WriteHeader(http.StatusNotFound)
		return
	}

	v := &validate.Request{
		R:  req,
		W:  w,
		ID: id,
	}

	if ok := v.Validate(level1, level2); !ok {
		return
	}

	item := hierarchy[label]
	res := &response.Response{
		ID:        item.ID,
		LabelCode: item.LabelCode,
		Label:     item.Label,
	}

	res.AddChildren(item.Children)
	if data := res.AddParent(item.Parents, label); data != nil {
		log.ErrorR(req, errors.New("too many parent elements found"), *data)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	res.AddLinks(req.URL.String())

	b, err := json.Marshal(res)
	if err != nil {
		log.ErrorR(req, err, nil)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}
