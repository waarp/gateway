package rest

import (
	"context"
	"net/http"
	"time"

	"code.waarp.fr/waarp/gateway-ng/pkg/log"
)

// This is the REST service interface
type Service struct {
	port   string
	logger *log.Logger
	server http.Server
}

// Signals that an error occurred when shutting down the service
type ShutdownError struct{ error }

// Starts the server listener, and restarts the service if an unexpected
// error occurred while listening
func listen(service *Service) {
	err := service.server.ListenAndServe()
	if err != http.ErrServerClosed {
		service.logger.Errorf("Unexpected REST service error: %s", err)
		service.logger.Info("Attempting to restart REST service.")

		service.StopRestService()
		service.StartRestService()
	}
}

// Starts the REST service
func (service *Service) StartRestService() {
	service.server = http.Server{Addr: service.port}

	status := &StatusHandler{logger: service.logger}

	http.Handle(STATUS_URI, status)

	go listen(service)

	service.logger.Info("REST service started.")
}

// Stops the REST service by first trying to shut down the service gracefully.
// If it fails after a 10 seconds delay, the service is forcefully stopped.
func (service *Service) StopRestService() {
	service.logger.Info("REST service shutdown initiated.")

	ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
	err := service.server.Shutdown(ctx)

	if err == nil && err != http.ErrServerClosed {
		service.logger.Warning("Failed to shutdown the REST service gracefully.")
		_ = service.server.Close()
		service.logger.Warning("The REST service was forcefully stopped.")
	} else {
		service.logger.Info("REST service shutdown complete.")
	}
}
