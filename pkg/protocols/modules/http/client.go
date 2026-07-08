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
	transporter httptransport.Transporter
	state       utils.State
}

func (h *httpClient) Name() string { return h.client.Name }

func (h *httpClient) Start() error {
	if h.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	if err := h.start(); err != nil {
		h.logger.Errorf("Failed to start HTTP client: %v", err)
		h.state.Set(utils.StateError, err.Error())
		snmp.ReportServiceFailure(h.client.Name, err)

		return err
	}

	h.state.Set(utils.StateRunning, "")

	return nil
}

func (h *httpClient) start() error {
	h.logger = logging.NewLogger(h.client.Name)

	if err := utils.JSONConvert(h.client.ProtoConfig, &h.conf); err != nil {
		h.logger.Errorf("Failed to parse the HTTP client's configuration: %v", err)

		return fmt.Errorf("failed to parse the HTTP client's configuration: %w", err)
	}

	var err error
	if h.transporter, err = httptransport.NewTransport(h.client.Protocol == HTTPS,
		h.client.LocalAddress.String()); err != nil {
		h.logger.Errorf("Failed to initialize the HTTP client's transport: %v", err)

		return fmt.Errorf("failed to initialize the HTTP client's transport: %w", err)
	}

	return nil
}

func (h *httpClient) InitTransfer(pip *pipeline.Pipeline) (protocol.TransferClient, *pipeline.Error) {
	transport, err := h.transporter.Get(pip)
	if err != nil {
		return nil, pipeline.NewErrorWith(err, types.TeInternal,
			"failed to initialize the HTTP client's transport")
	}

	return newTransferClient(pip, transport, h.client.Protocol == HTTPS), nil
}

func (h *httpClient) Stop(ctx context.Context) error {
	if !h.state.IsRunning() {
		return utils.ErrNotRunning
	}

	if err := pipeline.List.StopAllFromClient(ctx, h.client.ID); err != nil {
		h.logger.Errorf("Failed to interrupt HTTP client's running transfers: %v", err)
		h.state.Set(utils.StateError, err.Error())
		snmp.ReportServiceFailure(h.client.Name, err)

		return fmt.Errorf("failed to stop the HTTP client's running transfers: %w", err)
	}

	h.state.Set(utils.StateOffline, "")

	return nil
}

func (h *httpClient) State() (utils.StateCode, string) { return h.state.Get() }

func newTransferClient(pip *pipeline.Pipeline, transport *http.Transport, isHTTPS bool,
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
