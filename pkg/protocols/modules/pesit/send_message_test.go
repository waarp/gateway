package pesit

import (
	"context"
	"errors"
	"net"
	"testing"

	"code.waarp.fr/lib/pesit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/logging/logtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
)

var errNotImplemented = errors.New("not implemented")

func TestSendMessage(t *testing.T) {
	// Setup
	const (
		transferID = uint32(12345)
		message    = "Hello World!"
		password   = "sesame"
	)

	handler := makeTestMessageServer(t)
	ctx := gwtesting.NewTestClientCtx(t, Pesit, handler.addr, nil, nil)
	ctx.AddPassword(t, password)
	logger := logtest.GetTestLogger(t)

	clientService := newClient(ctx.DB, ctx.Client)
	require.NoError(t, clientService.Start())

	// Send message
	msgErr := sendInitialMessage(ctx.DB, logger, ctx.Partner, ctx.Account, transferID, message)
	require.NoError(t, msgErr)

	// Check credentials
	assert.Equal(t, ctx.Account.Login, handler.login)
	assert.Equal(t, password, handler.pswd)

	// Check message
	assert.NotNil(t, handler.msg)
	assert.Equal(t, transferID, handler.msg.TransferID)
	assert.Equal(t, message, handler.msg.Message)
}

type testMessageServer struct {
	addr        string
	login, pswd string
	msg         *pesit.MessageRequest
}

func (t *testMessageServer) Connect(c *pesit.ServerConnection) (pesit.TransferHandler, error) {
	t.login, t.pswd = c.ClientLogin(), c.ClientPassword()

	return t, nil
}
func (*testMessageServer) SelectFile(*pesit.ServerTransfer) error         { return errNotImplemented }
func (*testMessageServer) DeselectFile(error) error                       { return errNotImplemented }
func (*testMessageServer) OpenFile(*pesit.ServerTransfer) error           { return errNotImplemented }
func (*testMessageServer) CloseFile(error) error                          { return errNotImplemented }
func (*testMessageServer) StartDataTransfer(*pesit.ServerTransfer) error  { return errNotImplemented }
func (*testMessageServer) EndTransfer(*pesit.ServerTransfer, error) error { return errNotImplemented }
func (*testMessageServer) DataTransfer(*pesit.ServerTransfer) error       { return errNotImplemented }
func (*testMessageServer) Release(*pesit.ServerConnection)                {}

func (t *testMessageServer) HandleMessage(_ *pesit.ServerConnection, msg pesit.MessageRequest) error {
	t.msg = &msg

	return nil
}

func makeTestMessageServer(tb testing.TB) *testMessageServer {
	tb.Helper()

	handler := &testMessageServer{}
	serv := pesit.NewServer(handler)

	list, err := net.Listen("tcp", "localhost:0")
	require.NoError(tb, err)

	go serv.Serve(list)
	tb.Cleanup(func() { require.NoError(tb, serv.Close(context.Background())) })

	handler.addr = list.Addr().String()

	return handler
}
