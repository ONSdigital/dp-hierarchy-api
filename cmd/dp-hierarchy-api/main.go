package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/ONSdigital/dp-graph/graph"
	"github.com/ONSdigital/dp-hierarchy-api/api"
	"github.com/ONSdigital/dp-hierarchy-api/config"
	"github.com/ONSdigital/dp-hierarchy-api/models"
	"github.com/ONSdigital/go-ns/healthcheck"
	"github.com/ONSdigital/go-ns/log"
	"github.com/ONSdigital/go-ns/server"
	"github.com/gorilla/mux"
)

func main() {
	log.Namespace = "dp-hierarchy-api"

	config, err := config.Get()
	if err != nil {
		log.Error(err, nil)
		os.Exit(1)
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	// setup database
	graphDB, err := graph.NewHierarchyStore(context.Background())
	if err != nil {
		log.Error(err, nil)
		os.Exit(1)
	}

	// setup http server
	router := mux.NewRouter()
	router.Path("/healthcheck").HandlerFunc(healthcheck.Do)

	api.New(router, graphDB, config.HierarchyAPIURL)

	srv := server.New(config.BindAddr, router)
	srv.HandleOSSignals = false

	// put constants into model
	models.CodelistURL = config.CodelistAPIURL

	healthTicker := healthcheck.NewTicker(
		config.HealthCheckInterval,
		config.HealthCheckCriticalTimeout,
		graphDB,
	)

	// start http server
	httpServerDoneChan := make(chan error)
	go func() {
		log.Debug("starting http server", log.Data{"bind_addr": config.BindAddr})
		if err := srv.ListenAndServe(); err != nil {
			log.Error(err, nil)
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
	log.ErrorC("Start shutdown", err, logData)
	shutdownContext, shutdownContextCancel := context.WithTimeout(context.Background(), config.ShutdownTimeout)

	healthTicker.Close()

	go func() {
		if wantHTTPShutdown {
			if err := srv.Shutdown(shutdownContext); err != nil {
				log.ErrorC("error closing http server", err, nil)
			} else {
				log.Trace("http server shutdown", nil)
			}
		}

		if err := graphDB.Close(shutdownContext); err != nil {
			log.ErrorC("error closing db connection", err, nil)
		} else {
			log.Trace("db connection shutdown", nil)
		}

		shutdownContextCancel()
	}()

	// wait for timeout or success (cancel)
	<-shutdownContext.Done()

	log.Info("Shutdown done", log.Data{"context": shutdownContext.Err()})
	os.Exit(1)
}
