package amqp10

import (
	"context"
	"fmt"
	"path"
	"testing"
	"time"

	amqp "github.com/Azure/go-amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type amqp10ConfigChecker struct{}

func (amqp10ConfigChecker) IsValidProtocol(proto string) bool {
	return proto == AMQP10
}

func (amqp10ConfigChecker) CheckServerConfig(proto string, conf map[string]any) error {
	if proto != AMQP10 {
		return fmt.Errorf("unknown protocol %q", proto)
	}

	cfg := defaultServerConfig()
	if err := utils.JSONConvert(conf, cfg); err != nil {
		return err
	}

	return cfg.ValidServer()
}

func (amqp10ConfigChecker) CheckClientConfig(proto string, conf map[string]any) error {
	if proto != AMQP10 {
		return fmt.Errorf("unknown protocol %q", proto)
	}

	cfg := defaultClientConfig()
	if err := utils.JSONConvert(conf, cfg); err != nil {
		return err
	}

	return cfg.ValidClient()
}

func (amqp10ConfigChecker) CheckPartnerConfig(proto string, conf map[string]any) error {
	if proto != AMQP10 {
		return fmt.Errorf("unknown protocol %q", proto)
	}

	cfg := defaultPartnerConfig()
	if err := utils.JSONConvert(conf, cfg); err != nil {
		return err
	}

	return cfg.ValidPartner()
}

func setAMQP10ConfigChecker(t *testing.T) {
	t.Helper()

	oldChecker := model.ConfigChecker
	model.ConfigChecker = amqp10ConfigChecker{}
	t.Cleanup(func() { model.ConfigChecker = oldChecker })
}

func TestAMQP10ServerConsumesMessageToGatewayTransfer(t *testing.T) {
	setAMQP10ConfigChecker(t)
	db := dbtest.TestDatabase(t)
	rootDir := t.TempDir()

	agent := &model.LocalAgent{
		Name:     "amqp10-server",
		Protocol: AMQP10,
		Address:  types.Addr("localhost", 0),
		RootDir:  rootDir,
		ProtoConfig: map[string]any{
			"endpoint":         "amqps://broker.example.net:5671",
			"sourceAddress":    "gateway/in",
			"receiverLinkName": "gateway-amqp10-in",
			"localAccount":     "inbox",
			"ruleName":         "amqp10-inbound",
			"filenameTemplate": "${messageID}.txt",
		},
	}
	require.NoError(t, db.Insert(agent).Run())

	account := &model.LocalAccount{
		LocalAgentID: agent.ID,
		Login:        "inbox",
	}
	require.NoError(t, db.Insert(account).Run())

	rule := &model.Rule{
		Name:     "amqp10-inbound",
		IsSend:   false,
		Path:     "amqp10-inbound",
		LocalDir: "messages",
	}
	require.NoError(t, db.Insert(rule).Run())

	receiver := &fakeReceiver{err: context.Canceled}
	session := &fakeSession{receiver: receiver}
	conn := &fakeConnection{session: session}
	svc := newServer(db, agent)
	svc.dialer = fakeDialer{conn: conn}

	require.NoError(t, svc.Start())

	contentType := "application/octet-stream"
	msg := amqp.NewMessage([]byte("hello-amqp10-server"))
	msg.Properties = &amqp.MessageProperties{
		MessageID:     "msg-001",
		CorrelationID: "corr-001",
		ContentType:   &contentType,
	}
	require.NoError(t, svc.processMessage(msg))

	var historyEntries model.HistoryEntries
	require.NoError(t, db.Select(&historyEntries).OrderBy("id", false).Run())
	require.NotEmpty(t, historyEntries)
	history := historyEntries[0]
	content, err := fs.ReadFullFile(history.LocalPath)
	require.NoError(t, err)
	assert.Equal(t, "hello-amqp10-server", string(content))
	assert.Equal(t, "msg-001.txt", path.Base(history.LocalPath))

	stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, svc.Stop(stopCtx))
	assert.True(t, receiver.closed)
	assert.True(t, session.closed)
	assert.True(t, conn.closed)
}

func TestAMQP10ServerRejectsSendRule(t *testing.T) {
	setAMQP10ConfigChecker(t)
	db := dbtest.TestDatabase(t)

	agent := &model.LocalAgent{
		Name:     "amqp10-server-invalid",
		Protocol: AMQP10,
		Address:  types.Addr("localhost", 0),
		ProtoConfig: map[string]any{
			"endpoint":      "amqps://broker.example.net:5671",
			"sourceAddress": "gateway/in",
			"localAccount":  "inbox",
			"ruleName":      "amqp10-outbound",
		},
	}
	require.NoError(t, db.Insert(agent).Run())

	account := &model.LocalAccount{
		LocalAgentID: agent.ID,
		Login:        "inbox",
	}
	require.NoError(t, db.Insert(account).Run())

	rule := &model.Rule{
		Name:   "amqp10-outbound",
		IsSend: true,
		Path:   "amqp10-outbound",
	}
	require.NoError(t, db.Insert(rule).Run())

	svc := newServer(db, agent)
	svc.dialer = fakeDialer{conn: &fakeConnection{session: &fakeSession{receiver: &fakeReceiver{err: context.Canceled}}}}

	err := svc.Start()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "receive rule")
}
