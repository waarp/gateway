package pesit

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"testing"

	"code.waarp.fr/lib/pesit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/logtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
)

var errNotImplemented = errors.New("not implemented")

func TestSendMessage(t *testing.T) {
	// Setup
	const (
		transferID = uint32(12345)
		filename   = "test.pesit"
		message    = "Hello World!"
		password   = "sesame"
		customer   = "mr. customer"
		bank       = "bank institution"
	)

	handler := makeTestMessageServer(t)
	ctx := gwtesting.NewTestClientCtx(t, Pesit, handler.addr, nil, nil)
	ctx.AddPassword(t, password)
	logger := logtest.GetTestLogger(t,
		logtest.WithName("test_message_client"),
		logtest.WithLevel("TRACE"),
	)

	clientService := newClient(ctx.DB, ctx.Client)
	require.NoError(t, clientService.Start())

	infos := map[string]any{bankIDKey: bank, customerIDKey: customer}

	// Send message
	msgErr := sendInitialMessage(ctx.DB, logger, ctx.Partner, ctx.Account,
		infos, transferID, filename, strings.NewReader(message))
	require.NoError(t, msgErr)

	// Check credentials
	assert.Equal(t, ctx.Account.Login, handler.login)
	assert.Equal(t, password, handler.pswd)

	// Check message
	assert.Equal(t, pesit.FileACK, handler.msgType)
	assert.Equal(t, transferID, handler.msgTransID)
	assert.Equal(t, filename, handler.msgFilename)
	assert.Equal(t, customer, handler.msgCustomer)
	assert.Equal(t, bank, handler.msgBank)
	assert.Equal(t, message, handler.msgContent)
}

var _ pesit.MessageHandler = &testMessageServer{}

type testMessageServer struct {
	addr        string
	login, pswd string

	msgType     pesit.MessageType
	msgTransID  uint32
	msgCustomer string
	msgBank     string
	msgFilename string
	msgContent  string
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

func (t *testMessageServer) HandleMessage(req *pesit.MessageRequest, cont io.Reader,
) (*pesit.MessageResult, error) {
	t.msgType = req.MessageType
	t.msgTransID = req.TransferID
	t.msgFilename = req.Filename
	t.msgCustomer = req.CustomerID
	t.msgBank = req.BankID

	var err error
	if t.msgContent, err = utils.ReadString(cont); err != nil {
		err = pesit.NewDiagnostic(pesit.CodeUnresolvableIO, "failed to read message content")
	}

	return nil, err
}

func makeTestMessageServer(tb testing.TB) *testMessageServer {
	tb.Helper()

	logger := logtest.GetTestLogger(tb,
		logtest.WithName("test_message_server"),
		logtest.WithLevel("TRACE"),
	)
	handler := &testMessageServer{}
	serv := pesit.NewServer(handler)
	serv.Logger = logger.AsStdLogger(log.LevelDebug)
	serv.NetworkTrace = logger.AsStdLogger(log.LevelTrace)

	list, err := net.Listen("tcp", "localhost:0")
	require.NoError(tb, err)

	serv.Logger.Print("Starting test message server")

	go serv.Serve(list)
	tb.Cleanup(func() { require.NoError(tb, serv.Close(context.Background())) })

	handler.addr = list.Addr().String()

	return handler
}
