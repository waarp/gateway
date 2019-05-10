package rest

import (
	"context"
	"net/http"

	"code.waarp.fr/waarp/gateway-ng/pkg/log"
)

type Service struct {
	port   string
	logger *log.Logger
	server http.Server
}

func (service *Service) StartRestService() {
	service.server = http.Server{Addr: service.port}

	status := &StatusHandler{logger: service.logger}

	http.HandleFunc(STATUS_URI, status.ServeHTTP)

	go func() {
		if err := service.server.ListenAndServe(); err != http.ErrServerClosed {
			service.logger.Errorf("Unexpected REST service shutdown: %s", err)
		} else {
			service.logger.Info("REST service shutting down.")
		}
	}()
}

func (service *Service) stopRestService() {
	service.logger.Info("REST service shutdown initiated.")

	if err := service.server.Shutdown(context.Background()); err != nil {
		service.logger.Error("Failed to shutdown the REST service gracefully.")
	}

	service.logger.Info("REST service shutdown complete.")
}
