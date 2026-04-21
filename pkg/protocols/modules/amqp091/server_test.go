package amqp091

import (
	"context"
	"fmt"
	"path"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type amqp091ConfigChecker struct{}

func (amqp091ConfigChecker) IsValidProtocol(proto string) bool {
	return proto == AMQP091
}

func (amqp091ConfigChecker) CheckServerConfig(proto string, conf map[string]any) error {
	if proto != AMQP091 {
		return fmt.Errorf("unknown protocol %q", proto)
	}

	cfg := defaultServerConfig()
	if err := utils.JSONConvert(conf, cfg); err != nil {
		return err
	}

	return cfg.ValidServer()
}

func (amqp091ConfigChecker) CheckClientConfig(proto string, conf map[string]any) error {
	if proto != AMQP091 {
		return fmt.Errorf("unknown protocol %q", proto)
	}

	cfg := defaultClientConfig()
	if err := utils.JSONConvert(conf, cfg); err != nil {
		return err
	}

	return cfg.ValidClient()
}

func (amqp091ConfigChecker) CheckPartnerConfig(proto string, conf map[string]any) error {
	if proto != AMQP091 {
		return fmt.Errorf("unknown protocol %q", proto)
	}

	cfg := defaultPartnerConfig()
	if err := utils.JSONConvert(conf, cfg); err != nil {
		return err
	}

	return cfg.ValidPartner()
}

func setAMQP091ConfigChecker(t *testing.T) {
	t.Helper()

	oldChecker := model.ConfigChecker
	model.ConfigChecker = amqp091ConfigChecker{}
	t.Cleanup(func() { model.ConfigChecker = oldChecker })
}

type fakeAMQPDialer struct {
	conn amqpConnection
	err  error
}

func (f fakeAMQPDialer) DialConfig(string, *amqp.Config) (amqpConnection, error) {
	return f.conn, f.err
}

type fakeAMQPConnection struct {
	channel amqpChannel
	err     error
	closed  bool
}

func (f *fakeAMQPConnection) Channel() (amqpChannel, error) {
	return f.channel, f.err
}

func (f *fakeAMQPConnection) Close() error {
	f.closed = true

	return nil
}

type fakeAMQPChannel struct {
	deliveries chan amqp.Delivery
	closed     bool
	canceled   bool
}

func (*fakeAMQPChannel) ExchangeDeclare(string, string, bool, bool, bool, bool, amqp.Table) error {
	return nil
}

func (*fakeAMQPChannel) QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table,
) (amqp.Queue, error) {
	return amqp.Queue{Name: name}, nil
}

func (*fakeAMQPChannel) QueueBind(string, string, string, bool, amqp.Table) error {
	return nil
}

func (*fakeAMQPChannel) PublishWithContext(context.Context, string, string, bool, bool, amqp.Publishing) error {
	return nil
}

func (f *fakeAMQPChannel) Consume(string, string, bool, bool, bool, bool, amqp.Table) (<-chan amqp.Delivery, error) {
	return f.deliveries, nil
}

func (f *fakeAMQPChannel) Cancel(string, bool) error {
	f.canceled = true
	close(f.deliveries)

	return nil
}

func (*fakeAMQPChannel) Qos(int, int, bool) error { return nil }
func (*fakeAMQPChannel) Confirm(bool) error       { return nil }
func (*fakeAMQPChannel) NotifyPublish(confirm chan amqp.Confirmation) chan amqp.Confirmation {
	return confirm
}

func (f *fakeAMQPChannel) Close() error {
	f.closed = true

	return nil
}

func TestAMQP091ServerConsumesDeliveryToGatewayTransfer(t *testing.T) {
	setAMQP091ConfigChecker(t)
	db := dbtest.TestDatabase(t)
	rootDir := t.TempDir()

	agent := &model.LocalAgent{
		Name:     "amqp091-server",
		Protocol: AMQP091,
		Address:  types.Addr("localhost", 0),
		RootDir:  rootDir,
		ProtoConfig: map[string]any{
			"uri":              "amqp://broker.example.net:5672",
			"queue":            "gateway.in",
			"consumerTag":      "gateway-amqp091-in",
			"localAccount":     "inbox",
			"ruleName":         "amqp-inbound",
			"filenameHeader":   "filename",
			"filenameTemplate": "${messageID}.bin",
		},
	}
	require.NoError(t, db.Insert(agent).Run())

	account := &model.LocalAccount{
		LocalAgentID: agent.ID,
		Login:        "inbox",
	}
	require.NoError(t, db.Insert(account).Run())

	rule := &model.Rule{
		Name:     "amqp-inbound",
		IsSend:   false,
		Path:     "amqp-inbound",
		LocalDir: "messages",
	}
	require.NoError(t, db.Insert(rule).Run())

	channel := &fakeAMQPChannel{deliveries: make(chan amqp.Delivery, 1)}
	conn := &fakeAMQPConnection{channel: channel}
	svc := newServer(db, agent)
	svc.dialer = fakeAMQPDialer{conn: conn}

	require.NoError(t, svc.Start())

	delivery := amqp.Delivery{
		DeliveryTag: 1,
		MessageId:   "msg-001",
		Headers: amqp.Table{
			"filename": "inbound/report.txt",
		},
		Body: []byte("hello-amqp091"),
	}
	require.NoError(t, svc.processDelivery(&delivery))

	var historyEntries model.HistoryEntries
	require.NoError(t, db.Select(&historyEntries).OrderBy("id", false).Run())
	require.NotEmpty(t, historyEntries)
	history := historyEntries[0]
	content, err := fs.ReadFullFile(history.LocalPath)
	require.NoError(t, err)
	assert.Equal(t, "hello-amqp091", string(content))
	assert.Equal(t, "report.txt", path.Base(history.LocalPath))

	stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, svc.Stop(stopCtx))
	assert.True(t, channel.canceled)
	assert.True(t, channel.closed)
	assert.True(t, conn.closed)
}

func TestAMQP091ServerRejectsSendRule(t *testing.T) {
	setAMQP091ConfigChecker(t)
	db := dbtest.TestDatabase(t)

	agent := &model.LocalAgent{
		Name:     "amqp091-server-invalid",
		Protocol: AMQP091,
		Address:  types.Addr("localhost", 0),
		ProtoConfig: map[string]any{
			"uri":          "amqp://broker.example.net:5672",
			"queue":        "gateway.in",
			"localAccount": "inbox",
			"ruleName":     "amqp-outbound",
		},
	}
	require.NoError(t, db.Insert(agent).Run())

	account := &model.LocalAccount{
		LocalAgentID: agent.ID,
		Login:        "inbox",
	}
	require.NoError(t, db.Insert(account).Run())

	rule := &model.Rule{
		Name:   "amqp-outbound",
		IsSend: true,
		Path:   "amqp-outbound",
	}
	require.NoError(t, db.Insert(rule).Run())

	svc := newServer(db, agent)
	svc.dialer = fakeAMQPDialer{conn: &fakeAMQPConnection{channel: &fakeAMQPChannel{deliveries: make(chan amqp.Delivery)}}}

	err := svc.Start()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "receive rule")
}
