package main

import (
	"context"
	"errors"
	"github.com/ONSdigital/dp-api-clients-go/health"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	"os"
	"os/signal"
	"syscall"

	"github.com/ONSdigital/dp-graph/graph"
	"github.com/ONSdigital/dp-hierarchy-api/api"
	"github.com/ONSdigital/dp-hierarchy-api/config"
	"github.com/ONSdigital/dp-hierarchy-api/models"
	"github.com/ONSdigital/go-ns/server"
	"github.com/ONSdigital/log.go/log"
	"github.com/gorilla/mux"
)

var (
	// BuildTime represents the time in which the service was built
	BuildTime string
	// GitCommit represents the commit (SHA-1) hash of the service that is running
	GitCommit string
	// Version represents the version of the service that is running
	Version string
)

func main() {

	ctx := context.Background()
	log.Namespace = "dp-hierarchy-api"

	config, err := config.Get()
	if err != nil {
		log.Event(ctx, "error getting config", log.FATAL, log.Error(err))
		os.Exit(1)
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	// setup database
	graphDB, err := graph.NewHierarchyStore(ctx)
	if err != nil {
		log.Event(ctx, "error creating hierarchy store", log.FATAL, log.Error(err))
		os.Exit(1)
	}

	hc := startHealthCheck(ctx, config, graphDB)

	// setup http server
	router := mux.NewRouter()
	router.Path("/health").HandlerFunc(hc.Handler)

	api.New(router, graphDB, config.HierarchyAPIURL)

	srv := server.New(config.BindAddr, router)
	srv.HandleOSSignals = false

	// put constants into model
	models.CodelistURL = config.CodelistAPIURL

	// start http server
	httpServerDoneChan := make(chan error)
	go func() {
		log.Event(ctx, "starting http server", log.INFO, log.Data{"bind_addr": config.BindAddr})
		if err := srv.ListenAndServe(); err != nil {
			log.Event(ctx, "http server error", log.ERROR, log.Error(err))
		}
		close(httpServerDoneChan)
	}()

	// wait (indefinitely) for an exit event (either an OS signal or the httpServerDoneChan)
	// set `err` and logData
	wantHTTPShutdown := true
	logData := log.Data{}
	select {
	case sig := <-signals:
		err = errors.New("aborting after signal")
		logData["signal"] = sig.String()
	case err = <-httpServerDoneChan:
		wantHTTPShutdown = false
	}

	// gracefully shutdown the application, closing any open resources
	logData["timeout"] = config.ShutdownTimeout
	log.Event(ctx, "start shutdown", log.ERROR, log.Error(err), logData)
	shutdownContext, shutdownContextCancel := context.WithTimeout(context.Background(), config.ShutdownTimeout)

	hc.Stop()

	go func() {
		if wantHTTPShutdown {
			if err := srv.Shutdown(shutdownContext); err != nil {
				log.Event(ctx, "error closing http server", log.ERROR, log.Error(err))
			} else {
				log.Event(ctx, "http server shutdown", log.INFO)
			}
		}

		if err := graphDB.Close(shutdownContext); err != nil {
			log.Event(ctx, "error closing db connection", log.ERROR, log.Error(err))
		} else {
			log.Event(ctx, "db connection shutdown", log.INFO)
		}

		shutdownContextCancel()
	}()

	// wait for timeout or success (cancel)
	<-shutdownContext.Done()

	log.Event(ctx, "Shutdown done", log.INFO, log.Data{"context": shutdownContext.Err()})
	os.Exit(1)
}

func startHealthCheck(ctx context.Context, config *config.Config, graphDB *graph.DB) *healthcheck.HealthCheck {

	hasErrors := false
	versionInfo, err := healthcheck.NewVersionInfo(BuildTime, GitCommit, Version)
	if err != nil {
		log.Event(ctx, "error creating version info", log.FATAL, log.Error(err))
		hasErrors = true
	}

	hc := healthcheck.New(versionInfo, config.HealthCheckCriticalTimeout, config.HealthCheckInterval)

	if err = hc.AddCheck("Neo4J", graphDB.Checker); err != nil {
		hasErrors = true
		log.Event(nil, "error adding check for graph db", log.ERROR, log.Error(err))
	}

	codeListAPIHealthCheckClient := health.NewClient("Code List API", config.CodelistAPIURL)
	if err = hc.AddCheck("Code List API", codeListAPIHealthCheckClient.Checker); err != nil {
		log.Event(ctx, "error creating code list API health check", log.Error(err))
		hasErrors = true
	}

	if hasErrors {
		os.Exit(1)
	}

	hc.Start(ctx)

	return &hc
}
