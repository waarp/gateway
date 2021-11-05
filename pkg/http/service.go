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
	"time"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/proto"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/state"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
)

const readHeaderTimeout = 10 * time.Second

type httpService struct {
	logger  *log.Logger
	db      *database.DB
	agentID uint64
	state   state.State

	conf    config.HTTPProtoConfig
	serv    *http.Server
	running *service.TransferMap

	shutdown chan struct{}
}

// NewService initializes and returns a new HTTP service.
func NewService(db *database.DB, logger *log.Logger) proto.Service {
	return newService(db, logger)
}

func newService(db *database.DB, logger *log.Logger) *httpService {
	return &httpService{
		db:      db,
		logger:  logger,
		running: service.NewTransferMap(),
	}
}

func (h *httpService) Start(agent *model.LocalAgent) (err error) {
	h.agentID = agent.ID

	if code, _ := h.state.Get(); code != state.Offline && code != state.Error {
		h.logger.Info("Cannot start the server because it is already running.")

		return nil
	}

	defer func() {
		if err != nil {
			h.state.Set(state.Error, err.Error())
		}
	}()

	h.logger.Info("Starting HTTP server '%s'...", agent.Name)
	h.state.Set(state.Starting, "")

	if err2 := json.Unmarshal(agent.ProtoConfig, &h.conf); err2 != nil {
		h.logger.Error("Failed to parse server configuration: %s", err2)

		return fmt.Errorf("failed to parse server configuration: %w", err2)
	}

	var tlsConf *tls.Config

	if agent.Protocol == "https" {
		var err3 error
		if tlsConf, err3 = h.makeTLSConf(agent); err3 != nil {
			return err3
		}
	}

	//nolint:gosec //we cannot add a
	h.serv = &http.Server{
		Addr:              agent.Address,
		Handler:           h.makeHandler(),
		TLSConfig:         tlsConf,
		ErrorLog:          h.logger.AsStdLogger(log.LevelError),
		ReadHeaderTimeout: readHeaderTimeout,
	}

	if err4 := h.listen(agent); err4 != nil {
		return err4
	}

	h.logger.Info("HTTP server started at: %s", agent.Address)
	h.state.Set(state.Running, "")

	h.shutdown = make(chan struct{})

	return nil
}

func (h *httpService) Stop(ctx context.Context) error {
	if st, _ := h.state.Get(); st != state.Running {
		return nil
	}

	h.logger.Info("Shutdown command received...")
	h.state.Set(state.ShuttingDown, "")
	close(h.shutdown)

	if err := h.stop(ctx); err != nil {
		h.logger.Warning("Forcing service shutdown...")
		_ = h.serv.Close() //nolint:errcheck //error is irrelevant at this point
		h.state.Set(state.Error, err.Error())

		h.logger.Debug("Service was shut down forcefully.")

		return err
	}

	h.state.Set(state.Offline, "")
	h.logger.Info("HTTP server shutdown successful")

	return nil
}

func (h *httpService) stop(ctx context.Context) error {
	h.logger.Debug("Interrupting transfers...")

	if err := h.running.InterruptAll(ctx); err != nil {
		h.logger.Warning("Failed to interrupt R66 transfers: %s", err)

		return fmt.Errorf("could not halt the service gracefully: %w", err)
	}

	h.logger.Debug("Closing listener...")

	if err := h.serv.Shutdown(ctx); err != nil {
		h.logger.Warning("Error while closing HTTP listener: %s", err)
		h.state.Set(state.Offline, "")

		return fmt.Errorf("failed to stop the HTTP listener: %w", err)
	}

	return nil
}

func (h *httpService) State() *state.State {
	return &h.state
}

func (h *httpService) ManageTransfers() *service.TransferMap {
	return h.running
}
