package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/ONSdigital/dp-hierarchy-api/api"
	"github.com/ONSdigital/dp-hierarchy-api/config"
	"github.com/ONSdigital/dp-hierarchy-api/health"
	"github.com/ONSdigital/dp-hierarchy-api/models"
	"github.com/ONSdigital/dp-hierarchy-api/store"
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
	dbStore, err := store.New(config.DbAddr)
	if err != nil {
		log.Error(err, nil)
		os.Exit(1)
	}
	log.Debug("connected to db", nil)
	api.SetDatabase(dbStore)

	// setup http server
	router := mux.NewRouter()
	srv := server.New(config.BindAddr, router)
	srv.HandleOSSignals = false
	api.AddRoutes(router)

	// put config into api
	api.HierarchyAPIURL = config.HierarchyAPIURL

	// put constants into model
	models.CodelistURL = config.CodelistAPIURL

	// setup healthcheck
	health.SetDatabase(dbStore)
	health.AddRoutes(router)

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

	go func() {
		if wantHTTPShutdown {
			if err := srv.Shutdown(shutdownContext); err != nil {
				log.ErrorC("error closing http server", err, nil)
			} else {
				log.Trace("http server shutdown", nil)
			}
		}

		if err := dbStore.Close(shutdownContext); err != nil {
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
