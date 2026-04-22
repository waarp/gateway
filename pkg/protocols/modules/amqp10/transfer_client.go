package amqp10

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	amqp "github.com/Azure/go-amqp"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
)

var errCloseTransfer = errors.New("failed to close the AMQP 1.0 transfer resources")

type amqpDialer interface {
	Dial(ctx context.Context, addr string, opts *amqp.ConnOptions) (amqpConnection, error)
}

type amqpConnection interface {
	NewSession(ctx context.Context, opts *amqp.SessionOptions) (amqpSession, error)
	Close() error
}

type amqpSession interface {
	NewSender(ctx context.Context, target string, opts *amqp.SenderOptions) (amqpSender, error)
	NewReceiver(ctx context.Context, source string, opts *amqp.ReceiverOptions) (amqpReceiver, error)
	Close(ctx context.Context) error
}

type amqpSender interface {
	Send(ctx context.Context, msg *amqp.Message, opts *amqp.SendOptions) error
	Close(ctx context.Context) error
}

type amqpReceiver interface {
	Receive(ctx context.Context, opts *amqp.ReceiveOptions) (*amqp.Message, error)
	AcceptMessage(ctx context.Context, msg *amqp.Message) error
	ReleaseMessage(ctx context.Context, msg *amqp.Message) error
	Close(ctx context.Context) error
}

type defaultDialer struct{}

func (defaultDialer) Dial(ctx context.Context, addr string, opts *amqp.ConnOptions) (amqpConnection, error) {
	conn, err := amqp.Dial(ctx, addr, opts)
	if err != nil {
		return nil, fmt.Errorf("dial AMQP 1.0 broker: %w", err)
	}

	return amqpConnWrapper{Conn: conn}, nil
}

type amqpConnWrapper struct{ *amqp.Conn }

func (c amqpConnWrapper) NewSession(ctx context.Context, opts *amqp.SessionOptions) (amqpSession, error) {
	session, err := c.Conn.NewSession(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("open AMQP 1.0 session: %w", err)
	}

	return amqpSessionWrapper{Session: session}, nil
}

type amqpSessionWrapper struct{ *amqp.Session }

func (s amqpSessionWrapper) NewSender(
	ctx context.Context,
	target string,
	opts *amqp.SenderOptions,
) (amqpSender, error) {
	sender, err := s.Session.NewSender(ctx, target, opts)
	if err != nil {
		return nil, fmt.Errorf("open AMQP 1.0 sender: %w", err)
	}

	return amqpSenderWrapper{Sender: sender}, nil
}

func (s amqpSessionWrapper) NewReceiver(
	ctx context.Context,
	source string,
	opts *amqp.ReceiverOptions,
) (amqpReceiver, error) {
	receiver, err := s.Session.NewReceiver(ctx, source, opts)
	if err != nil {
		return nil, fmt.Errorf("open AMQP 1.0 receiver: %w", err)
	}

	return amqpReceiverWrapper{Receiver: receiver}, nil
}

type amqpSenderWrapper struct{ *amqp.Sender }

type amqpReceiverWrapper struct{ *amqp.Receiver }

type transferClient struct {
	pip     *pipeline.Pipeline
	conf    clientConfig
	partner partnerConfig
	dialer  amqpDialer
	timeout time.Duration

	conn     amqpConnection
	session  amqpSession
	sender   amqpSender
	receiver amqpReceiver
	message  *amqp.Message
}

func (t *transferClient) Request() *pipeline.Error {
	ctx, cancel := context.WithTimeout(context.Background(), t.timeout)
	defer cancel()

	conn, err := t.dialer.Dial(ctx, t.conf.Endpoint, &amqp.ConnOptions{
		ContainerID: connectionName(t.conf.ConnectionName, t.pip.TransCtx.Transfer.RemoteTransferID),
		IdleTimeout: time.Duration(t.conf.IdleTimeoutSeconds) * time.Second,
	})
	if err != nil {
		return pipeline.NewErrorWith(types.TeConnection, "failed to connect to the AMQP 1.0 broker", err)
	}
	session, err := conn.NewSession(ctx, nil)
	if err != nil {
		_ = conn.Close()
		return pipeline.NewErrorWith(types.TeConnection, "failed to open the AMQP 1.0 session", err)
	}

	t.conn = conn
	t.session = session

	if t.pip.TransCtx.Rule.IsSend {
		sender, senderErr := session.NewSender(ctx, t.conf.TargetAddress, nil)
		if senderErr != nil {
			return t.closeWith(types.TeConnection, "failed to open the AMQP 1.0 sender", senderErr)
		}
		t.sender = sender
	} else {
		receiver, receiverErr := session.NewReceiver(ctx, t.partner.SourceAddress, nil)
		if receiverErr != nil {
			return t.closeWith(types.TeConnection, "failed to open the AMQP 1.0 receiver", receiverErr)
		}
		t.receiver = receiver
	}

	return nil
}

func (t *transferClient) Send(file protocol.SendFile) *pipeline.Error {
	if t.sender == nil {
		return pipeline.NewError(types.TeInternal, "the AMQP 1.0 sender is not initialized")
	}

	payload, err := io.ReadAll(file)
	if err != nil {
		return pipeline.NewErrorWith(types.TeDataTransfer, "failed to read the AMQP 1.0 payload", err)
	}

	msg := amqp.NewMessage(payload)
	now := time.Now().UTC()
	contentType := "application/octet-stream"
	msg.Properties = &amqp.MessageProperties{
		MessageID:     t.pip.TransCtx.Transfer.RemoteTransferID,
		CorrelationID: t.pip.TransCtx.Transfer.RemoteTransferID,
		ContentType:   &contentType,
		CreationTime:  &now,
	}
	msg.ApplicationProperties = map[string]any{
		"waarpTransferID":   t.pip.TransCtx.Transfer.RemoteTransferID,
		"waarpRuleName":     t.pip.TransCtx.Rule.Name,
		"waarpSrcFilename":  t.pip.TransCtx.Transfer.SrcFilename,
		"waarpDestFilename": t.pip.TransCtx.Transfer.DestFilename,
	}

	ctx, cancel := context.WithTimeout(context.Background(), t.timeout)
	defer cancel()
	sendErr := t.sender.Send(ctx, msg, nil)
	if sendErr != nil {
		return t.closeWith(types.TeDataTransfer, "failed to send the AMQP 1.0 message", sendErr)
	}

	t.pip.TransCtx.Transfer.Filesize = int64(len(payload))

	return nil
}

func (t *transferClient) Receive(file protocol.ReceiveFile) *pipeline.Error {
	if t.receiver == nil {
		return pipeline.NewError(types.TeInternal, "the AMQP 1.0 receiver is not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), t.timeout)
	defer cancel()
	msg, err := t.receiver.Receive(ctx, nil)
	if err != nil {
		return pipeline.NewErrorWith(types.TeConnection, "failed to receive the AMQP 1.0 message", err)
	}

	t.message = msg
	data := msg.GetData()
	_, writeErr := file.Write(data)
	if writeErr != nil {
		return pipeline.NewErrorWith(types.TeDataTransfer, "failed to write the AMQP 1.0 payload locally", writeErr)
	}

	t.pip.TransCtx.Transfer.Filesize = int64(len(data))
	if msg.Properties != nil {
		if id, ok := msg.Properties.MessageID.(string); ok && strings.TrimSpace(id) != "" {
			t.pip.TransCtx.Transfer.RemoteTransferID = id
		}
	}

	info := t.pip.TransCtx.Transfer.TransferInfo
	if info == nil {
		info = map[string]any{}
		t.pip.TransCtx.Transfer.TransferInfo = info
	}
	if msg.Properties != nil {
		info["amqp10MessageID"] = stringifyAMQPValue(msg.Properties.MessageID)
		info["amqp10CorrelationID"] = stringifyAMQPValue(msg.Properties.CorrelationID)
	}
	info["amqp10SourceAddress"] = t.partner.SourceAddress

	return nil
}

func (t *transferClient) EndTransfer() *pipeline.Error {
	if t.message != nil && t.receiver != nil {
		ctx, cancel := context.WithTimeout(context.Background(), t.timeout)
		defer cancel()
		if err := t.receiver.AcceptMessage(ctx, t.message); err != nil {
			return t.closeWith(types.TeFinalization, "failed to acknowledge the AMQP 1.0 message", err)
		}
	}
	if err := t.close(); err != nil {
		return pipeline.NewErrorWith(types.TeFinalization, "failed to close the AMQP 1.0 transfer", err)
	}

	return nil
}

func (t *transferClient) SendError(_ types.TransferErrorCode, _ string) {
	if t.message != nil && t.receiver != nil {
		ctx, cancel := context.WithTimeout(context.Background(), t.timeout)
		defer cancel()
		releaseErr := t.receiver.ReleaseMessage(ctx, t.message)
		if releaseErr != nil {
			_ = releaseErr
		}
	}
	_ = t.close()
}

func (t *transferClient) closeWith(code types.TransferErrorCode, msg string, cause error) *pipeline.Error {
	_ = t.close()
	if cause == nil {
		return pipeline.NewError(code, msg)
	}

	return pipeline.NewErrorWith(code, msg, cause)
}

func (t *transferClient) close() error {
	var errs []string
	ctx, cancel := context.WithTimeout(context.Background(), t.timeout)
	defer cancel()
	if t.sender != nil {
		if err := t.sender.Close(ctx); err != nil {
			errs = append(errs, err.Error())
		}
		t.sender = nil
	}
	if t.receiver != nil {
		if err := t.receiver.Close(ctx); err != nil {
			errs = append(errs, err.Error())
		}
		t.receiver = nil
	}
	if t.session != nil {
		if err := t.session.Close(ctx); err != nil {
			errs = append(errs, err.Error())
		}
		t.session = nil
	}
	if t.conn != nil {
		if err := t.conn.Close(); err != nil {
			errs = append(errs, err.Error())
		}
		t.conn = nil
	}
	if len(errs) != 0 {
		return fmt.Errorf("%w: %s", errCloseTransfer, strings.Join(errs, "; "))
	}

	return nil
}

func connectionName(base, transferID string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "waarp-amqp10"
	}
	if strings.TrimSpace(transferID) == "" {
		return base
	}

	return base + "-" + transferID
}

func stringifyAMQPValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprint(typed)
	}
}
