package amqp10

import (
	"bytes"
	"context"
	"errors"
	"hash"
	"testing"

	amqp "github.com/Azure/go-amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

type fakeReceiveFile struct {
	bytes.Buffer
}

func (f *fakeReceiveFile) WriteAt(p []byte, off int64) (int, error) {
	data := f.Buffer.Bytes()
	if int(off) > len(data) {
		padding := make([]byte, int(off)-len(data))
		_, _ = f.Buffer.Write(padding)
	}
	data = f.Buffer.Bytes()
	end := int(off) + len(p)
	if end > len(data) {
		grow := make([]byte, end-len(data))
		_, _ = f.Buffer.Write(grow)
		data = f.Buffer.Bytes()
	}
	copy(data[int(off):end], p)
	return len(p), nil
}

func (f *fakeReceiveFile) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case 0:
		return offset, nil
	case 1:
		return int64(f.Buffer.Len()) + offset, nil
	case 2:
		return int64(f.Buffer.Len()) + offset, nil
	default:
		return 0, errors.New("invalid whence")
	}
}

func (*fakeReceiveFile) CheckHash(hash.Hash, []byte) error { return nil }

type fakeDialer struct {
	conn amqpConnection
	err  error
}

func (f fakeDialer) Dial(context.Context, string, *amqp.ConnOptions) (amqpConnection, error) {
	return f.conn, f.err
}

type fakeConnection struct {
	session amqpSession
	err     error
	closed  bool
}

func (f *fakeConnection) NewSession(context.Context, *amqp.SessionOptions) (amqpSession, error) {
	return f.session, f.err
}

func (f *fakeConnection) Close() error {
	f.closed = true
	return nil
}

type fakeSession struct {
	sender   amqpSender
	receiver amqpReceiver
	err      error
	closed   bool
}

func (f *fakeSession) NewSender(context.Context, string, *amqp.SenderOptions) (amqpSender, error) {
	return f.sender, f.err
}

func (f *fakeSession) NewReceiver(context.Context, string, *amqp.ReceiverOptions) (amqpReceiver, error) {
	return f.receiver, f.err
}

func (f *fakeSession) Close(context.Context) error {
	f.closed = true
	return nil
}

type fakeSender struct {
	message *amqp.Message
	closed  bool
	err     error
}

func (f *fakeSender) Send(_ context.Context, msg *amqp.Message, _ *amqp.SendOptions) error {
	f.message = msg
	return f.err
}

func (f *fakeSender) Close(context.Context) error {
	f.closed = true
	return nil
}

type fakeReceiver struct {
	message  *amqp.Message
	accepted bool
	released bool
	closed   bool
	err      error
}

func (f *fakeReceiver) Receive(context.Context, *amqp.ReceiveOptions) (*amqp.Message, error) {
	return f.message, f.err
}

func (f *fakeReceiver) AcceptMessage(context.Context, *amqp.Message) error {
	f.accepted = true
	return nil
}

func (f *fakeReceiver) ReleaseMessage(context.Context, *amqp.Message) error {
	f.released = true
	return nil
}

func (f *fakeReceiver) Close(context.Context) error {
	f.closed = true
	return nil
}

func makePipeline(isSend bool) *pipeline.Pipeline {
	return &pipeline.Pipeline{
		Logger: logging.NewLogger("amqp10-test"),
		TransCtx: &model.TransferContext{
			Transfer: &model.Transfer{
				RemoteTransferID: "rt-123",
				SrcFilename:      "src.txt",
				DestFilename:     "dst.txt",
			},
			Rule:        &model.Rule{Name: "rule-a", IsSend: isSend},
			RemoteAgent: &model.RemoteAgent{},
		},
	}
}

func TestTransferClientSendPublishesAMQP10Message(t *testing.T) {
	sender := &fakeSender{}
	session := &fakeSession{sender: sender}
	conn := &fakeConnection{session: session}
	tx := &transferClient{
		pip: makePipeline(true),
		conf: clientConfig{
			Endpoint:      "amqps://broker.example.net:5671",
			TargetAddress: "gateway/out",
		},
		partner: partnerConfig{},
		dialer:  fakeDialer{conn: conn},
	}

	require.Nil(t, tx.Request())
	require.Nil(t, tx.Send(bytes.NewReader([]byte("hello-amqp10"))))
	require.NotNil(t, sender.message)
	assert.Equal(t, []byte("hello-amqp10"), sender.message.GetData())
	require.NotNil(t, sender.message.Properties)
	assert.Equal(t, "rt-123", sender.message.Properties.MessageID)
	assert.Equal(t, int64(12), tx.pip.TransCtx.Transfer.Filesize)
	require.Nil(t, tx.EndTransfer())
	assert.True(t, sender.closed)
	assert.True(t, session.closed)
	assert.True(t, conn.closed)
}

func TestTransferClientReceiveConsumesAMQP10Message(t *testing.T) {
	contentType := "application/octet-stream"
	msg := amqp.NewMessage([]byte("hello-amqp10"))
	msg.Properties = &amqp.MessageProperties{
		MessageID:     "msg-001",
		CorrelationID: "corr-001",
		ContentType:   &contentType,
	}
	receiver := &fakeReceiver{message: msg}
	session := &fakeSession{receiver: receiver}
	conn := &fakeConnection{session: session}
	out := &fakeReceiveFile{}

	tx := &transferClient{
		pip: makePipeline(false),
		conf: clientConfig{
			Endpoint:      "amqps://broker.example.net:5671",
			TargetAddress: "gateway/out",
		},
		partner: partnerConfig{
			SourceAddress: "gateway/in",
		},
		dialer: fakeDialer{conn: conn},
	}

	require.Nil(t, tx.Request())
	require.Nil(t, tx.Receive(out))
	assert.Equal(t, "hello-amqp10", out.String())
	assert.Equal(t, "msg-001", tx.pip.TransCtx.Transfer.RemoteTransferID)
	require.Nil(t, tx.EndTransfer())
	assert.True(t, receiver.accepted)
	assert.True(t, receiver.closed)
}

func TestTransferClientRequestPropagatesDialError(t *testing.T) {
	tx := &transferClient{
		pip: makePipeline(true),
		conf: clientConfig{
			Endpoint:      "amqps://broker.example.net:5671",
			TargetAddress: "gateway/out",
		},
		dialer: fakeDialer{err: errors.New("boom")},
	}

	perr := tx.Request()
	require.NotNil(t, perr)
	assert.Contains(t, perr.Error(), "failed to connect")
}
