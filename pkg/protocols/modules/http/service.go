// Package http contains the functions necessary to execute a file transfer
// using the HTTP protocol. The package defines both a client and a server for
// HTTP.
package http

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/analytics"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const readHeaderTimeout = 10 * time.Second

type httpService struct {
	db    *database.DB
	agent *model.LocalAgent

	logger *log.Logger
	state  utils.State
	conf   httpsServerConfig
	serv   *http.Server

	tracer   func() pipeline.Trace
	shutdown chan struct{}
}

func (h *httpService) Name() string { return h.agent.Name }

func (h *httpService) reportError(err error) {
	if err == nil {
		return
	}

	h.logger.Error(err.Error())
	h.state.Set(utils.StateError, err.Error())
	snmp.ReportServiceFailure(h.agent.Name, err)
}

func (h *httpService) Start() (retErr error) {
	if h.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	h.logger = logging.NewLogger(h.agent.Name)
	defer h.reportError(retErr)

	if err := h.db.Get(h.agent, "id=?", h.agent.ID).Run(); err != nil {
		return fmt.Errorf("failed to retrieve the HTTP agent: %w", err)
	}

	h.logger = logging.NewLogger(h.agent.Name)
	h.logger.Info("Starting HTTP server...")

	if err := h.start(); err != nil {
		return err
	}

	h.state.Set(utils.StateRunning, "")
	h.logger.Infof("HTTP server started successfully on %q", h.serv.Addr)

	return nil
}

func (h *httpService) start() error {
	if err := utils.JSONConvert(h.agent.ProtoConfig, &h.conf); err != nil {
		return fmt.Errorf("failed to parse server configuration: %w", err)
	}

	h.serv = &http.Server{
		Handler:           h.makeHandler(),
		ErrorLog:          h.logger.AsStdLogger(log.LevelError),
		ReadHeaderTimeout: readHeaderTimeout,
		ConnState: func(_ net.Conn, state http.ConnState) {
			switch state {
			case http.StateNew:
				analytics.AddIncomingConnection()
			case http.StateClosed:
				analytics.SubIncomingConnection()
			default:
			}
		},
	}

	if err := h.listen(); err != nil {
		return err
	}

	h.shutdown = make(chan struct{})

	return nil
}

func (h *httpService) Stop(ctx context.Context) (retErr error) {
	if !h.state.IsRunning() {
		return utils.ErrNotRunning
	}

	h.logger.Info("Stopping HTTP server...")
	defer h.reportError(retErr)
	defer h.serv.Close() //nolint:errcheck // error does not matter at this point
	close(h.shutdown)

	if err := pipeline.List.StopAllFromServer(ctx, h.agent.ID); err != nil {
		return fmt.Errorf("could not halt the service gracefully: %w", err)
	}

	if err := h.serv.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to stop the HTTP listener: %w", err)
	}

	h.state.Set(utils.StateOffline, "")
	h.logger.Info("HTTP server stopped successfully")

	return nil
}

func (h *httpService) State() (utils.StateCode, string)         { return h.state.Get() }
func (h *httpService) SetTracer(getTrace func() pipeline.Trace) { h.tracer = getTrace }
