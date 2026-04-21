package amqp091

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
)

var errCloseTransfer = errors.New("failed to close the AMQP 0.9.1 transfer resources")

type amqpDialer interface {
	DialConfig(uri string, cfg *amqp.Config) (amqpConnection, error)
}

type amqpConnection interface {
	Channel() (amqpChannel, error)
	Close() error
}

type amqpChannel interface {
	ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error
	QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error)
	QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error
	PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
	Consume(
		queue, consumer string,
		autoAck, exclusive, noLocal, noWait bool,
		args amqp.Table,
	) (<-chan amqp.Delivery, error)
	Cancel(consumer string, noWait bool) error
	Qos(prefetchCount, prefetchSize int, global bool) error
	Confirm(noWait bool) error
	NotifyPublish(confirm chan amqp.Confirmation) chan amqp.Confirmation
	Close() error
}

type defaultDialer struct{}

func (defaultDialer) DialConfig(uri string, cfg *amqp.Config) (amqpConnection, error) {
	conn, err := amqp.DialConfig(uri, *cfg)
	if err != nil {
		return nil, fmt.Errorf("dial AMQP 0.9.1 broker: %w", err)
	}

	return amqpConnWrapper{Connection: conn}, nil
}

type amqpConnWrapper struct{ *amqp.Connection }

func (c amqpConnWrapper) Channel() (amqpChannel, error) {
	ch, err := c.Connection.Channel()
	if err != nil {
		return nil, fmt.Errorf("open AMQP 0.9.1 channel: %w", err)
	}

	return ch, nil
}

type transferClient struct {
	pip     *pipeline.Pipeline
	conf    clientConfig
	partner partnerConfig
	dialer  amqpDialer
	timeout time.Duration

	conn       amqpConnection
	channel    amqpChannel
	deliveries <-chan amqp.Delivery
	delivery   *amqp.Delivery
}

func (t *transferClient) Request() *pipeline.Error {
	cfg := amqp.Config{
		Heartbeat: time.Duration(t.conf.HeartbeatSeconds) * time.Second,
		Properties: amqp.Table{
			"connection_name": connectionName(t.conf.ConnectionName, t.pip.TransCtx.Transfer.RemoteTransferID),
		},
	}

	conn, err := t.dialer.DialConfig(t.conf.URI, &cfg)
	if err != nil {
		return pipeline.NewErrorWith(types.TeConnection, "failed to connect to the AMQP 0.9.1 broker", err)
	}
	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return pipeline.NewErrorWith(types.TeConnection, "failed to open the AMQP 0.9.1 channel", err)
	}

	t.conn = conn
	t.channel = channel

	if !t.pip.TransCtx.Rule.IsSend {
		if t.partner.PrefetchCount > 0 {
			qosErr := t.channel.Qos(t.partner.PrefetchCount, 0, false)
			if qosErr != nil {
				return t.closeWith(types.TeConnection, "failed to configure the AMQP 0.9.1 consumer QoS", qosErr)
			}
		}
		deliveries, consumeErr := t.channel.Consume(
			t.partner.Queue,
			t.partner.ConsumerTag,
			t.partner.AutoAck,
			false,
			false,
			false,
			nil,
		)
		if consumeErr != nil {
			return t.closeWith(types.TeConnection, "failed to start the AMQP 0.9.1 consumer", consumeErr)
		}

		t.deliveries = deliveries
	}

	return nil
}

func (t *transferClient) Send(file protocol.SendFile) *pipeline.Error {
	payload, err := io.ReadAll(file)
	if err != nil {
		return pipeline.NewErrorWith(types.TeDataTransfer, "failed to read the AMQP 0.9.1 payload", err)
	}

	publishing := amqp.Publishing{
		MessageId:     t.pip.TransCtx.Transfer.RemoteTransferID,
		CorrelationId: t.pip.TransCtx.Transfer.RemoteTransferID,
		Timestamp:     time.Now().UTC(),
		ContentType:   "application/octet-stream",
		Body:          payload,
		Type:          "waarp.transfer",
	}
	if t.conf.PersistentMessages {
		publishing.DeliveryMode = amqp.Persistent
	}

	key := renderRoutingKey(t.conf.RoutingKeyTemplate, t.pip)
	ctx, cancel := context.WithTimeout(context.Background(), t.timeout)
	defer cancel()

	if t.conf.PublisherConfirms {
		confirmErr := t.channel.Confirm(false)
		if confirmErr != nil {
			return t.closeWith(types.TeConnection, "failed to enable AMQP 0.9.1 publisher confirms", confirmErr)
		}
	}

	publishErr := t.channel.PublishWithContext(
		ctx,
		firstNonEmpty(t.partner.Exchange, t.conf.Exchange),
		key,
		t.conf.Mandatory,
		false,
		publishing,
	)
	if publishErr != nil {
		return t.closeWith(types.TeDataTransfer, "failed to publish the AMQP 0.9.1 message", publishErr)
	}

	if t.conf.PublisherConfirms {
		select {
		case confirmation := <-t.channel.NotifyPublish(make(chan amqp.Confirmation, 1)):
			if !confirmation.Ack {
				return t.closeWith(types.TeDataTransfer, "the AMQP 0.9.1 broker negatively acknowledged the message", nil)
			}
		case <-ctx.Done():
			return t.closeWith(types.TeConnection, "timed out waiting for the AMQP 0.9.1 broker confirmation", ctx.Err())
		}
	}

	t.pip.TransCtx.Transfer.Filesize = int64(len(payload))

	return nil
}

func (t *transferClient) Receive(file protocol.ReceiveFile) *pipeline.Error {
	if t.deliveries == nil {
		return pipeline.NewError(types.TeInternal, "the AMQP 0.9.1 consumer is not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), t.timeout)
	defer cancel()

	select {
	case delivery, ok := <-t.deliveries:
		if !ok {
			return pipeline.NewError(types.TeConnection, "the AMQP 0.9.1 consumer channel closed unexpectedly")
		}
		t.delivery = &delivery
		if _, err := file.Write(delivery.Body); err != nil {
			return pipeline.NewErrorWith(types.TeDataTransfer, "failed to write the AMQP 0.9.1 payload locally", err)
		}
		t.pip.TransCtx.Transfer.Filesize = int64(len(delivery.Body))
		if delivery.MessageId != "" {
			t.pip.TransCtx.Transfer.RemoteTransferID = delivery.MessageId
		}
		info := t.pip.TransCtx.Transfer.TransferInfo
		if info == nil {
			info = map[string]any{}
			t.pip.TransCtx.Transfer.TransferInfo = info
		}
		info["amqpExchange"] = delivery.Exchange
		info["amqpRoutingKey"] = delivery.RoutingKey
		info["amqpMessageID"] = delivery.MessageId

		return nil
	case <-ctx.Done():
		return pipeline.NewErrorWith(types.TeConnection, "timed out waiting for the AMQP 0.9.1 message", ctx.Err())
	}
}

func (t *transferClient) EndTransfer() *pipeline.Error {
	if t.delivery != nil && !t.partner.AutoAck {
		if err := t.delivery.Ack(false); err != nil {
			return t.closeWith(types.TeFinalization, "failed to acknowledge the AMQP 0.9.1 message", err)
		}
	}
	if err := t.close(); err != nil {
		return pipeline.NewErrorWith(types.TeFinalization, "failed to close the AMQP 0.9.1 transfer", err)
	}

	return nil
}

func (t *transferClient) SendError(_ types.TransferErrorCode, _ string) {
	if t.delivery != nil && !t.partner.AutoAck {
		if nackErr := t.delivery.Nack(false, true); nackErr != nil {
			_ = nackErr
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
	if t.channel != nil {
		if err := t.channel.Close(); err != nil {
			errs = append(errs, err.Error())
		}
		t.channel = nil
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
		base = "waarp-amqp091"
	}
	if strings.TrimSpace(transferID) == "" {
		return base
	}

	return base + "-" + transferID
}

func renderRoutingKey(template string, pip *pipeline.Pipeline) string {
	out := strings.TrimSpace(template)
	if out == "" {
		return pip.TransCtx.Transfer.RemoteTransferID
	}

	replacements := map[string]string{
		"${transferID}":   pip.TransCtx.Transfer.RemoteTransferID,
		"${ruleName}":     pip.TransCtx.Rule.Name,
		"${srcFilename}":  pip.TransCtx.Transfer.SrcFilename,
		"${destFilename}": pip.TransCtx.Transfer.DestFilename,
	}
	for key, value := range replacements {
		out = strings.ReplaceAll(out, key, value)
	}

	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}

	return ""
}
