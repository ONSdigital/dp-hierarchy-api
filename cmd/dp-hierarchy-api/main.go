package main

import (
	"context"
	"errors"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/ONSdigital/dp-api-clients-go/v2/health"
	"github.com/ONSdigital/dp-graph/v2/graph"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	"github.com/ONSdigital/dp-hierarchy-api/api"
	"github.com/ONSdigital/dp-hierarchy-api/config"
	"github.com/ONSdigital/dp-hierarchy-api/models"
	dphttp "github.com/ONSdigital/dp-net/v2/http"
	"github.com/ONSdigital/log.go/v2/log"
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
		log.Fatal(ctx, "error getting config", err)
		os.Exit(1)
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	// setup database
	graphDB, err := graph.NewHierarchyStore(ctx)
	if err != nil {
		log.Fatal(ctx, "error creating hierarchy store", err)
		os.Exit(1)
	}

	graphErrorConsumer := graph.NewLoggingErrorConsumer(ctx, graphDB.Errors)

	hc := startHealthCheck(ctx, config, graphDB)

	// setup http server
	router := mux.NewRouter()
	router.Path("/health").HandlerFunc(hc.Handler)

	// store URLs using net/url URL type to prevent error checking in handlers
	hierarchyAPIURL, err := url.Parse(config.HierarchyAPIURL)
	if err != nil {
		log.Fatal(ctx, "error parsing hierarchy API URL", err, log.Data{"url": config.HierarchyAPIURL})
		os.Exit(1)
	}

	codeListAPIURL, err := url.Parse(config.CodelistAPIURL)
	if err != nil {
		log.Fatal(ctx, "error parsing codeList API URL", err, log.Data{"url": config.CodelistAPIURL})
		os.Exit(1)
	}

	// check if URLRewriting is enabled
	enableURLRewriting := config.EnableURLRewriting
	if enableURLRewriting {
		log.Info(ctx, "URL rewriting enabled")
	}

	api.New(router, graphDB, hierarchyAPIURL, codeListAPIURL, enableURLRewriting)

	srv := dphttp.NewServer(config.BindAddr, router)
	srv.HandleOSSignals = false

	// put constants into model
	models.CodelistURL = config.CodelistAPIURL

	// start http server
	httpServerDoneChan := make(chan error)
	go func() {
		log.Info(ctx, "starting http server", log.Data{"bind_addr": config.BindAddr})
		if err := srv.ListenAndServe(); err != nil {
			log.Error(ctx, "http server error", err)
		}
		close(httpServerDoneChan)
	}()

	// wait (indefinitely) for an exit event (either an OS signal or the httpServerDoneChan)
	// set `err` and logData
	wantHTTPShutdown := true
	logData := log.Data{}
	select {
	case sig := <-signals:
		logData["signal"] = sig.String()
		log.Info(ctx, "aborting after signal", logData)
	case <-httpServerDoneChan:
		wantHTTPShutdown = false
	}

	// gracefully shutdown the application, closing any open resources
	logData["timeout"] = config.ShutdownTimeout
	log.Info(ctx, "start shutdown", logData)
	shutdownContext, shutdownContextCancel := context.WithTimeout(context.Background(), config.ShutdownTimeout)
	hasShutdownError := false

	go func() {

		log.Info(ctx, "stopping health checks")
		hc.Stop()

		if wantHTTPShutdown {
			log.Info(ctx, "stopping http server")
			if err := srv.Shutdown(shutdownContext); err != nil {
				log.Error(ctx, "error closing http server", err)
				hasShutdownError = true
			}
		}

		log.Info(ctx, "closing graph db connection")
		if err := graphDB.Close(shutdownContext); err != nil {
			log.Error(ctx, "error closing db connection", err)
			hasShutdownError = true
		}

		log.Info(ctx, "closing graph db error consumer")
		if err := graphErrorConsumer.Close(shutdownContext); err != nil {
			log.Error(ctx, "error closing graph db error consumer", err)
			hasShutdownError = true
		}

		shutdownContextCancel()
	}()

	// wait for timeout or success (cancel)
	<-shutdownContext.Done()

	if hasShutdownError {
		err = errors.New("failed to shutdown gracefully")
		log.Error(ctx, "failed to shutdown gracefully ", err)
		os.Exit(1)
	}

	log.Info(ctx, "graceful shutdown was successful")
	os.Exit(0)
}

func startHealthCheck(ctx context.Context, config *config.Config, graphDB *graph.DB) *healthcheck.HealthCheck {

	hasErrors := false
	versionInfo, err := healthcheck.NewVersionInfo(BuildTime, GitCommit, Version)
	if err != nil {
		log.Fatal(ctx, "error creating version info", err)
		hasErrors = true
	}

	hc := healthcheck.New(versionInfo, config.HealthCheckCriticalTimeout, config.HealthCheckInterval)

	if err = hc.AddCheck("Graph DB", graphDB.Checker); err != nil {
		hasErrors = true
		log.Error(ctx, "error adding check for graph db", err)
	}

	codeListAPIHealthCheckClient := health.NewClient("Code List API", config.CodelistAPIURL)
	if err = hc.AddCheck("Code List API", codeListAPIHealthCheckClient.Checker); err != nil {
		log.Error(ctx, "error creating code list API health check", err)
		hasErrors = true
	}

	if hasErrors {
		os.Exit(1)
	}

	hc.Start(ctx)

	return &hc
}
