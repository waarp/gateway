package protocolstest

import (
	"context"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type agent interface {
	database.Identifier
	database.GetBean
	GetName() string
}

type (
	TestServer = TestService[*model.LocalAgent]
	TestClient = TestService[*model.Client]
)

func NewTestClient(db database.ReadAccess, agent *model.Client) *TestClient {
	return &TestClient{db: db, agent: agent}
}

func NewTestServer(db database.ReadAccess, agent *model.LocalAgent) *TestServer {
	return &TestServer{db: db, agent: agent}
}

type TestService[T agent] struct {
	db    database.ReadAccess
	agent T
	state utils.State
}

func (t *TestService[T]) Name() string { return t.agent.GetName() }

func (t *TestService[T]) Start() error {
	if t.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	if err := t.db.Get(t.agent, "id=?", t.agent.GetID()).Run(); err != nil {
		t.state.Set(utils.StateError, err.Error())

		return fmt.Errorf("failed to retrieve server: %w", err)
	}

	t.state.Set(utils.StateRunning, "")

	return nil
}

func (t *TestService[T]) Stop(context.Context) error {
	if !t.state.IsRunning() {
		return utils.ErrNotRunning
	}

	t.state.Set(utils.StateOffline, "")

	return nil
}

func (t *TestService[T]) State() (utils.StateCode, string) { return t.state.Get() }
func (t *TestService[T]) InitTransfer(*pipeline.Pipeline) (protocol.TransferClient, *pipeline.Error) {
	return TestTransferClient{}, nil
}

type TestTransferClient struct{}

func (TestTransferClient) Request() *pipeline.Error                     { return nil }
func (TestTransferClient) Send(protocol.SendFile) *pipeline.Error       { return nil }
func (TestTransferClient) Receive(protocol.ReceiveFile) *pipeline.Error { return nil }
func (TestTransferClient) EndTransfer() *pipeline.Error                 { return nil }
func (TestTransferClient) SendError(types.TransferErrorCode, string)    {}
