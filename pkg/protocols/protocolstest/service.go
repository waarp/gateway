package protocolstest

import (
	"context"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type TestService struct{ state utils.State }

func (t *TestService) Start() error {
	if t.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	t.state.Set(utils.StateRunning, "")

	return nil
}

func (t *TestService) Stop(context.Context) error {
	if !t.state.IsRunning() {
		return utils.ErrNotRunning
	}

	t.state.Set(utils.StateOffline, "")

	return nil
}

func (t *TestService) State() (utils.StateCode, string) { return t.state.Get() }
func (t *TestService) InitTransfer(*pipeline.Pipeline) (protocol.TransferClient, *pipeline.Error) {
	return TestTransferClient{}, nil
}

type TestTransferClient struct{}

func (TestTransferClient) Request() *pipeline.Error                     { return nil }
func (TestTransferClient) Send(protocol.SendFile) *pipeline.Error       { return nil }
func (TestTransferClient) Receive(protocol.ReceiveFile) *pipeline.Error { return nil }
func (TestTransferClient) EndTransfer() *pipeline.Error                 { return nil }
func (TestTransferClient) SendError(types.TransferErrorCode, string)    {}
