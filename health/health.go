package health

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ONSdigital/dp-hierarchy-api/models"
	"github.com/ONSdigital/go-ns/log"
	"github.com/gorilla/mux"
)

type healthMessage struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

var (
	HealthChannel chan error
	dbStore       models.Storer
)

// AddRoutes is called by main to add the routers served by this
func AddRoutes(r *mux.Router) {
	r.Path("/healthcheck").HandlerFunc(handler)
}

// SetDatabase sets the Storer interface for this package
func SetDatabase(db models.Storer) {
	dbStore = db
}

func handler(w http.ResponseWriter, r *http.Request) {
	var (
		healthIssue string
		err         error
	)

	// assume all well
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	body := []byte(`{"status":"OK"}`) // quicker than json.Marshal(healthMessage{...})

	// test db access
	if err := dbStore.Ping(r.Context()); err != nil {
		healthIssue = err.Error()
	}

	// when there's a healthIssue, change headers and content
	if healthIssue != "" {
		w.WriteHeader(http.StatusInternalServerError)
		if body, err = json.Marshal(healthMessage{
			Status: "error",
			Error:  healthIssue,
		}); err != nil {
			log.Error(err, nil)
			panic(err)
		}
	}

	// return json
	fmt.Fprintf(w, string(body))
}
