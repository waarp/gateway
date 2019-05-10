package rest

import (
	"context"
	"net/http"
	"time"

	"code.waarp.fr/waarp/gateway-ng/pkg/conf"
	"code.waarp.fr/waarp/gateway-ng/pkg/log"
)

// This is the REST service interface
type Service struct {
	Logger *log.Logger
	server http.Server
	config conf.ServerConfig
}

// Signals that an error occurred when shutting down the service
type ShutdownError struct{ error }

// Starts the server listener, and restarts the service if an unexpected
// error occurred while listening
func listen(service *Service) {
	err := service.server.ListenAndServe()
	if err != http.ErrServerClosed {
		service.Logger.Errorf("Unexpected REST service error: %s", err)
		service.Logger.Info("Attempting to restart REST service.")

		service.StopRestService()
		service.StartRestService(service.config)
	}
}

// Starts the REST service
func (service *Service) StartRestService(config conf.ServerConfig) {
	service.server = http.Server{Addr: ":" + config.Rest.Port}

	status := &statusHandler{logger: service.Logger}

	http.Handle(STATUS_URI, status)

	go listen(service)

	service.Logger.Info("REST service started.")
}

// Stops the REST service by first trying to shut down the service gracefully.
// If it fails after a 10 seconds delay, the service is forcefully stopped.
func (service *Service) StopRestService() {
	service.Logger.Info("REST service shutdown initiated.")

	ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
	err := service.server.Shutdown(ctx)

	if err != nil && err != http.ErrServerClosed {
		service.Logger.Warningf("Failed to shutdown the REST service gracefully : %s", err)
		_ = service.server.Close()
		service.Logger.Warning("The REST service was forcefully stopped.")
	} else {
		service.Logger.Info("REST service shutdown complete.")
	}
}
