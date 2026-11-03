package pesit

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"strings"
	"testing"

	"code.waarp.fr/lib/pesit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/analytics"
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
	msgErr := sendInitialMessage(ctx.DB, logger, ctx.Partner, ctx.Account, ctx.RulePush,
		infos, transferID, filename, strings.NewReader(message))
	require.NoError(t, msgErr)

	// Check pre-connection
	assert.True(t, handler.receivedPreConn)
	assert.Equal(t, "        ", handler.preConnLogin)
	assert.Equal(t, "        ", handler.preConnPswd)

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

func TestSendMessageNonStandard(t *testing.T) {
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
	ctx := gwtesting.NewTestClientCtx(t, Pesit, handler.addr, nil, map[string]any{
		"compatibilityMode": CompatibilityModeNonStandard,
	})
	ctx.AddPassword(t, password)
	logger := logtest.GetTestLogger(t,
		logtest.WithName("test_message_client"),
		logtest.WithLevel("TRACE"),
	)

	clientService := newClient(ctx.DB, ctx.Client)
	require.NoError(t, clientService.Start())

	infos := map[string]any{bankIDKey: bank, customerIDKey: customer}

	// Send message
	msgErr := sendInitialMessage(ctx.DB, logger, ctx.Partner, ctx.Account, ctx.RulePush,
		infos, transferID, filename, strings.NewReader(message))
	require.NoError(t, msgErr)

	// Check pre-connection
	assert.True(t, handler.receivedPreConn)
	assert.Equal(t, "        ", handler.preConnLogin)
	assert.Equal(t, "        ", handler.preConnPswd)

	// Check credentials
	assert.Equal(t, ctx.Account.Login, handler.login)
	assert.Equal(t, password, handler.pswd)

	// Check message
	assert.Equal(t, pesit.FileACK, handler.msgType)
	assert.Equal(t, transferID, handler.msgTransID)
	assert.Equal(t, ctx.RulePush.Name, handler.msgFilename)
	assert.Equal(t, customer, handler.msgCustomer)
	assert.Equal(t, bank, handler.msgBank)
	assert.Equal(t, message, handler.msgContent)
}

// The message sent in TLS opens a counted connection (#827): the analytics
// service sees one open outgoing connection while the message is handled,
// none once the message has been sent.
func TestSendMessageTLS(t *testing.T) {
	const (
		transferID = uint32(12345)
		filename   = "test.pesit"
		message    = "Hello World!"
		password   = "sesame"
	)

	handler := makeTestMessageServerTLS(t)
	ctx := gwtesting.NewTestClientCtx(t, PesitTLS, handler.addr, nil, nil)
	ctx.AddPassword(t, password)
	ctx.AddCert(t, gwtesting.ServerCertPEM)
	logger := logtest.GetTestLogger(t,
		logtest.WithName("test_message_client_tls"),
		logtest.WithLevel("TRACE"),
	)

	analytics.GlobalService = &analytics.Service{DB: ctx.DB}
	t.Cleanup(func() { analytics.GlobalService = nil })

	clientService := newClient(ctx.DB, ctx.Client)
	require.NoError(t, clientService.Start())

	msgErr := sendInitialMessage(ctx.DB, logger, ctx.Partner, ctx.Account, ctx.RulePush,
		map[string]any{}, transferID, filename, strings.NewReader(message))
	require.NoError(t, msgErr)

	assert.Equal(t, message, handler.msgContent)
	assert.EqualValues(t, 1, handler.outgoingConns, "the connection should be counted while the message is handled")
	assert.Zero(t, analytics.GlobalService.OpenOutgoingConnections.Load(),
		"the connection should be counted off once the message is sent")
}

var _ pesit.MessageHandler = &testMessageServer{}

type testMessageServer struct {
	addr        string
	login, pswd string

	outgoingConns int64 // open outgoing connections of the client when the message is handled

	receivedPreConn           bool
	preConnLogin, preConnPswd string

	msgType     pesit.MessageType
	msgTransID  uint32
	msgCustomer string
	msgBank     string
	msgFilename string
	msgContent  string
}

func (t *testMessageServer) ValidatePreConnection(login string, password string) error {
	t.receivedPreConn, t.preConnLogin, t.preConnPswd = true, login, password

	return nil
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

	if analytics.GlobalService != nil {
		t.outgoingConns = analytics.GlobalService.OpenOutgoingConnections.Load()
	}

	var err error
	if t.msgContent, err = utils.ReadString(cont); err != nil {
		err = pesit.NewDiagnostic(pesit.CodeUnresolvableIO, "failed to read message content")
	}

	return nil, err
}

func makeTestMessageServer(tb testing.TB) *testMessageServer {
	tb.Helper()

	list, err := net.Listen("tcp", "localhost:0")
	require.NoError(tb, err)

	return serveTestMessages(tb, list)
}

func makeTestMessageServerTLS(tb testing.TB) *testMessageServer {
	tb.Helper()

	list, err := tls.Listen("tcp", "localhost:0", &tls.Config{
		Certificates: []tls.Certificate{gwtesting.ServerCert},
		MinVersion:   tls.VersionTLS12,
	})
	require.NoError(tb, err)

	return serveTestMessages(tb, list)
}

func serveTestMessages(tb testing.TB, list net.Listener) *testMessageServer {
	tb.Helper()

	logger := logtest.GetTestLogger(tb,
		logtest.WithName("test_message_server"),
		logtest.WithLevel("INFO"),
	)
	handler := &testMessageServer{}
	serv := pesit.NewServer(handler)
	serv.ValidatePreConnection = handler.ValidatePreConnection
	serv.Logger = logger.AsStdLogger(log.LevelDebug)
	serv.NetworkTrace = logger.AsStdLogger(log.LevelTrace)

	serv.Logger.Print("Starting test message server")

	go serv.Serve(list)
	tb.Cleanup(func() { require.NoError(tb, serv.Close(context.Background())) })

	handler.addr = list.Addr().String()

	return handler
}
