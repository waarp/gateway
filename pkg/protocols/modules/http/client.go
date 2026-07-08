package http

import (
	"context"
	"fmt"
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/http/httptransport"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	schemeHTTP  = "http://"
	schemeHTTPS = "https://"
)

var (
	errPause    = pipeline.NewError(types.TeStopped, "transfer paused by remote host")
	errShutdown = pipeline.NewError(types.TeShuttingDown, "remote host is shutting down")
	errCancel   = pipeline.NewError(types.TeCanceled, "transfer canceled by remote host")
)

type httpClient struct {
	db     *database.DB
	client *model.Client
	conf   httpsClientConfig

	logger      *log.Logger
	transporter *httptransport.Transporter
	state       utils.State
}

func (h *httpClient) Name() string { return h.client.Name }

func (h *httpClient) reportError(err error) {
	if err == nil {
		return
	}

	h.logger.Error(err.Error())
	h.state.Set(utils.StateError, err.Error())
	snmp.ReportServiceFailure(h.client.Name, err)
}

func (h *httpClient) Start() (retErr error) {
	if h.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	h.logger = logging.NewLogger(h.client.Name)
	defer h.reportError(retErr)

	if err := h.db.Get(h.client, "id=?", h.client.ID).Run(); err != nil {
		return fmt.Errorf("failed to retrieve the HTTP client: %w", err)
	}

	h.logger = logging.NewLogger(h.client.Name)
	h.logger.Info("Starting HTTP client...")

	if err := utils.JSONConvert(h.client.ProtoConfig, &h.conf); err != nil {
		return fmt.Errorf("failed to parse the HTTP client's configuration: %w", err)
	}

	var err error
	if h.transporter, err = httptransport.NewTransporter(h.client.Protocol == HTTPS,
		h.client.LocalAddress.String(), h.db.Config.Overrides); err != nil {
		return fmt.Errorf("failed to initialize the HTTP client's transport: %w", err)
	}

	h.state.Set(utils.StateRunning, "")
	h.logger.Info("HTTP client started successfully")

	return nil
}

func (h *httpClient) InitTransfer(pip *pipeline.Pipeline) (protocol.TransferClient, *pipeline.Error) {
	transport := h.transporter.Connect(pip)

	return newTransferClient(pip, transport, h.client.Protocol == HTTPS), nil
}

func (h *httpClient) Stop(ctx context.Context) (retErr error) {
	if !h.state.IsRunning() {
		return utils.ErrNotRunning
	}

	defer h.reportError(retErr)
	defer h.transporter.Close()

	h.logger.Info("Stopping HTTP client...")

	if err := pipeline.List.StopAllFromClient(ctx, h.client.ID); err != nil {
		return fmt.Errorf("failed to stop the HTTP client's running transfers: %w", err)
	}

	h.state.Set(utils.StateOffline, "")
	h.logger.Info("HTTP client stopped successfully")

	return nil
}

func (h *httpClient) State() (utils.StateCode, string) { return h.state.Get() }

func newTransferClient(pip *pipeline.Pipeline, transport http.RoundTripper, isHTTPS bool,
) protocol.TransferClient {
	client := &http.Client{Transport: transport}
	scheme := schemeHTTP
	if isHTTPS {
		scheme = schemeHTTPS
	}

	if pip.TransCtx.Rule.IsSend {
		return &postClient{
			pip:    pip,
			client: client,
			scheme: scheme,
			reqErr: make(chan error),
			resp:   make(chan *http.Response),
		}
	}

	return &getClient{
		pip:    pip,
		client: client,
		scheme: scheme,
	}
}
