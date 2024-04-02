package http

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const schemeHTTP = "http://"

var (
	errPause    = pipeline.NewError(types.TeStopped, "transfer paused by remote host")
	errShutdown = pipeline.NewError(types.TeShuttingDown, "remote host is shutting down")
	errCancel   = pipeline.NewError(types.TeCanceled, "transfer canceled by remote host")
)

type httpClient struct {
	db     *database.DB
	client *model.Client

	logger    *log.Logger
	transport *http.Transport
	state     utils.State
}

func (h *httpClient) Start() error {
	if h.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	if err := h.start(); err != nil {
		h.logger.Error("Failed to start HTTP client: %s", err)
		h.state.Set(utils.StateError, err.Error())

		return err
	}

	h.state.Set(utils.StateRunning, "")

	return nil
}

func (h *httpClient) start() error {
	h.logger = logging.NewLogger(h.client.Name)
	h.transport = &http.Transport{}

	if h.client.LocalAddress != "" {
		localAddr, err := net.ResolveTCPAddr("tcp", h.client.LocalAddress)
		if err != nil {
			h.logger.Error("Failed to parse the HTTP client's local address: %v", err)

			return fmt.Errorf("failed to parse the HTTP client's local address: %w", err)
		}

		h.transport.DialContext = (&net.Dialer{LocalAddr: localAddr}).DialContext
	}

	return nil
}

func (h *httpClient) InitTransfer(pip *pipeline.Pipeline) (protocol.TransferClient, *pipeline.Error) {
	return newTransferClient(pip, h.transport, false), nil
}

func (h *httpClient) Stop(ctx context.Context) error {
	if !h.state.IsRunning() {
		return utils.ErrNotRunning
	}

	if err := pipeline.List.StopAllFromClient(ctx, h.client.ID); err != nil {
		h.logger.Error("Failed to interrupt HTTP client's running transfers: %v", err)
		h.state.Set(utils.StateError, err.Error())

		return fmt.Errorf("failed to stop the HTTP client's running transfers: %w", err)
	}

	h.state.Set(utils.StateOffline, "")

	return nil
}

func (h *httpClient) State() (utils.StateCode, string) { return h.state.Get() }

func newTransferClient(pip *pipeline.Pipeline, transport *http.Transport, isHTTPS bool,
) protocol.TransferClient {
	if pip.TransCtx.Rule.IsSend {
		return &postClient{
			pip:       pip,
			transport: transport,
			isHTTPS:   isHTTPS,
			reqErr:    make(chan error),
			resp:      make(chan *http.Response),
		}
	}

	return &getClient{
		pip:       pip,
		transport: transport,
		isHTTPS:   isHTTPS,
	}
}
