// Package http contains the functions necessary to execute a file transfer
// using the HTTP protocol. The package defines both a client and a server for
// HTTP.
package http

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"

	"code.bcarlin.xyz/go/logging"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/gatewayd"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
)

func init() {
	gatewayd.ServiceConstructors["http"] = NewService
}

type httpService struct {
	logger *log.Logger
	db     *database.DB
	agent  *model.LocalAgent
	state  service.State

	conf    config.HTTPProtoConfig
	serv    *http.Server
	running *service.TransferMap

	shutdown chan struct{}
}

// NewService initializes and returns a new HTTP service.
func NewService(db *database.DB, agent *model.LocalAgent, logger *log.Logger) service.ProtoService {
	return &httpService{
		logger:   logger,
		db:       db,
		agent:    agent,
		running:  service.NewTransferMap(),
		shutdown: make(chan struct{}),
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
		return err
	}

	var tlsConf *tls.Config
	if h.conf.UseHTTPS {
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

	return nil
}

func (h *httpService) Stop(ctx context.Context) error {
	h.logger.Info("Shutdown command received...")
	if state, _ := h.state.Get(); state != service.Running {
		h.logger.Info("Cannot stop because the server is not running")
		return nil
	}

	h.state.Set(service.ShuttingDown, "")

	if h.shutdown == nil {
		h.shutdown = make(chan struct{})
	}
	close(h.shutdown)

	if err := h.running.InterruptAll(ctx); err != nil {
		h.logger.Error("Failed to interrupt R66 transfers, forcing exit")
		return ctx.Err()
	}

	h.logger.Debug("Closing listener...")
	_ = h.serv.Shutdown(ctx)

	h.state.Set(service.Offline, "")
	return nil
}

func (h *httpService) State() *service.State {
	panic("implement me")
}

func (h *httpService) ManageTransfers() *service.TransferMap {
	return h.running
}
