// Package http contains the functions necessary to execute a file transfer
// using the HTTP protocol. The package defines both a client and a server for
// HTTP.
package http

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"

	"code.bcarlin.xyz/go/logging"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/service"
)

type httpService struct {
	logger *log.Logger
	db     *database.DB
	agent  *model.LocalAgent
	state  *service.State

	conf    config.HTTPProtoConfig
	serv    *http.Server
	running *service.TransferMap

	shutdown chan struct{}
}

// NewService initializes and returns a new HTTP service.
func NewService(db *database.DB, agent *model.LocalAgent, logger *log.Logger) service.ProtoService {
	return &httpService{
		logger:  logger,
		db:      db,
		agent:   agent,
		running: service.NewTransferMap(),
		state:   &service.State{},
	}
}

func (h *httpService) Start() error {
	h.logger.Info("Starting HTTP service...")

	if state, _ := h.state.Get(); state != service.Offline && state != service.Error {
		h.logger.Infof("Cannot start because the server is already running.")

		return nil
	}

	h.state.Set(service.Starting, "")

	if err := json.Unmarshal(h.agent.ProtoConfig, &h.conf); err != nil {
		h.logger.Errorf("Failed to parse server configuration: %s", err)
		h.state.Set(service.Error, "failed to parse server configuration")

		return fmt.Errorf("failed to parse server configuration: %w", err)
	}

	var tlsConf *tls.Config

	if h.agent.Protocol == "https" {
		var err error
		if tlsConf, err = h.makeTLSConf(); err != nil {
			h.state.Set(service.Error, err.Error())

			return err
		}
	}

	h.serv = &http.Server{
		Addr:      h.agent.Address,
		Handler:   h.makeHandler(),
		TLSConfig: tlsConf,
		ErrorLog:  h.logger.AsStdLog(logging.ERROR),
	}

	if err := h.listen(); err != nil {
		h.state.Set(service.Error, err.Error())

		return err
	}

	h.logger.Infof("HTTP server started at: %s", h.agent.Address)
	h.state.Set(service.Running, "")

	h.shutdown = make(chan struct{})

	return nil
}

func (h *httpService) Stop(ctx context.Context) error {
	if state, _ := h.state.Get(); state != service.Running {
		return nil
	}

	h.logger.Info("Shutdown command received...")
	h.state.Set(service.ShuttingDown, "")
	close(h.shutdown)

	if err := h.stop(ctx); err != nil {
		h.logger.Warningf("Forcing service shutdown...")
		_ = h.serv.Close() //nolint:errcheck //error is irrelevant at this point
		h.state.Set(service.Error, err.Error())

		h.logger.Debug("Service was shut down forcefully.")

		return err
	}

	h.state.Set(service.Offline, "")
	h.logger.Info("HTTP server shutdown successful")

	return nil
}

func (h *httpService) stop(ctx context.Context) error {
	h.logger.Debug("Interrupting transfers...")

	if err := h.running.InterruptAll(ctx); err != nil {
		h.logger.Warningf("Failed to interrupt R66 transfers: %s", err)

		return fmt.Errorf("could not halt the service gracefully: %w", err)
	}

	h.logger.Debug("Closing listener...")

	if err := h.serv.Shutdown(ctx); err != nil {
		h.logger.Warningf("Error while closing HTTP listener: %s", err)

		return fmt.Errorf("failed to stop the HTTP listener: %w", err)
	}

	return nil
}

func (h *httpService) State() *service.State {
	return h.state
}

func (h *httpService) ManageTransfers() *service.TransferMap {
	return h.running
}
