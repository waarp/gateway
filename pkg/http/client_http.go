package http

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/state"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

var (
	errPause    = types.NewTransferError(types.TeStopped, "transfer paused by remote host")
	errShutdown = types.NewTransferError(types.TeShuttingDown, "remote host is shutting down")
	errCancel   = types.NewTransferError(types.TeCanceled, "transfer canceled by remote host")
)

type httpClient struct {
	transfers *service.TransferMap
	client    *model.Client
	transport *http.Transport
	state     state.State
}

func NewHTTPClient(dbClient *model.Client) (pipeline.Client, error) {
	cli := &httpClient{
		transfers: service.NewTransferMap(),
		client:    dbClient,
		transport: &http.Transport{},
	}

	if dbClient.LocalAddress != "" {
		localAddr, err := net.ResolveTCPAddr("tcp", dbClient.LocalAddress)
		if err != nil {
			return nil, fmt.Errorf("failed to parse the HTTP client's local address: %w", err)
		}

		cli.transport.DialContext = (&net.Dialer{LocalAddr: localAddr}).DialContext
	}

	return cli, nil
}

func (h *httpClient) Start() error {
	h.state.Set(state.Running, "")

	return nil
}

func (h *httpClient) State() *state.State { return &h.state }

func (h *httpClient) ManageTransfers() *service.TransferMap { return h.transfers }

func (h *httpClient) InitTransfer(pip *pipeline.Pipeline) (pipeline.TransferClient, *types.TransferError) {
	return newTransferClient(pip, h.transport, false), nil
}

func (h *httpClient) Stop(ctx context.Context) error {
	if err := h.transfers.InterruptAll(ctx); err != nil {
		return fmt.Errorf("failed to stop the running transfers: %w", err)
	}

	return nil
}

func newTransferClient(pip *pipeline.Pipeline, transport *http.Transport,
	isHTTPS bool,
) pipeline.TransferClient {
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
