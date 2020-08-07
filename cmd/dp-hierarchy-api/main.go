package main

import (
	"context"
	"errors"
	"github.com/ONSdigital/dp-api-clients-go/health"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	"os"
	"os/signal"
	"syscall"

	"github.com/ONSdigital/dp-graph/v2/graph"
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
	log.Event(ctx, "start shutdown", log.INFO, logData)
	shutdownContext, shutdownContextCancel := context.WithTimeout(context.Background(), config.ShutdownTimeout)
	hasShutdownError := false

	go func() {

		log.Event(ctx, "stopping health checks", log.INFO)
		hc.Stop()
		log.Event(ctx, "health checks stopped", log.INFO)

		if wantHTTPShutdown {
			log.Event(ctx, "stopping http server", log.INFO)
			if err := srv.Shutdown(shutdownContext); err != nil {
				log.Event(ctx, "error closing http server", log.ERROR, log.Error(err))
				hasShutdownError = true
			} else {
				log.Event(ctx, "http server shutdown", log.INFO)
			}
		}

		log.Event(ctx, "closing graph db connection", log.INFO)
		if err := graphDB.Close(shutdownContext); err != nil {
			log.Event(ctx, "error closing db connection", log.ERROR, log.Error(err))
			hasShutdownError = true
		} else {
			log.Event(ctx, "graph db connection closed", log.INFO)
		}

		shutdownContextCancel()
	}()

	// wait for timeout or success (cancel)
	<-shutdownContext.Done()

	if hasShutdownError {
		err = errors.New("failed to shutdown gracefully")
		log.Event(ctx, "failed to shutdown gracefully ", log.ERROR, log.Error(err))
		os.Exit(1)
	}

	log.Event(ctx, "graceful shutdown was successful", log.INFO)
	os.Exit(0)
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
