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

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/analytics"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
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

func (h *httpService) Start() error {
	if h.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	if err := h.start(); err != nil {
		h.state.Set(utils.StateError, err.Error())
		snmp.ReportServiceFailure(h.agent.Name, err)

		return err
	}

	h.state.Set(utils.StateRunning, "")

	return nil
}

func (h *httpService) start() error {
	h.logger = logging.NewLogger(h.agent.Name)
	h.logger.Info("Starting HTTP server...")

	if err := utils.JSONConvert(h.agent.ProtoConfig, &h.conf); err != nil {
		h.logger.Errorf("Failed to parse server configuration: %v", err)

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
	// h.serv.SetKeepAlivesEnabled(false)

	if err := h.listen(); err != nil {
		return err
	}

	h.logger.Infof("HTTP server started successfully on %q", h.serv.Addr)

	h.shutdown = make(chan struct{})

	return nil
}

func (h *httpService) Stop(ctx context.Context) error {
	if !h.state.IsRunning() {
		return utils.ErrNotRunning
	}

	if err := h.stop(ctx); err != nil {
		h.logger.Notice("Forcing service shutdown...")
		_ = h.serv.Close() //nolint:errcheck //error is irrelevant at this point
		h.logger.Notice("Server was shut down forcefully.")

		h.state.Set(utils.StateError, err.Error())
		snmp.ReportServiceFailure(h.agent.Name, err)

		return err
	}

	h.state.Set(utils.StateOffline, "")

	return nil
}

func (h *httpService) stop(ctx context.Context) error {
	h.logger.Debug("Interrupting transfers...")

	if err := pipeline.List.StopAllFromServer(ctx, h.agent.ID); err != nil {
		h.logger.Errorf("Failed to interrupt R66 transfers: %v", err)

		return fmt.Errorf("could not halt the service gracefully: %w", err)
	}

	h.logger.Debug("Closing listener...")

	if err := h.serv.Shutdown(ctx); err != nil {
		h.logger.Errorf("Error while closing HTTP listener: %v", err)

		return fmt.Errorf("failed to stop the HTTP listener: %w", err)
	}

	h.logger.Info("HTTP server shutdown successful")

	return nil
}

func (h *httpService) State() (utils.StateCode, string)         { return h.state.Get() }
func (h *httpService) SetTracer(getTrace func() pipeline.Trace) { h.tracer = getTrace }
