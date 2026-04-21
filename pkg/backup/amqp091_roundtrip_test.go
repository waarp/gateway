package backup

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backupfile "code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const backupAMQP091 = "amqp091"

type backupAMQP091ConfigChecker struct{}

func (backupAMQP091ConfigChecker) IsValidProtocol(proto string) bool { return proto == backupAMQP091 }

func (backupAMQP091ConfigChecker) CheckServerConfig(proto string, conf map[string]any) error {
	if proto != backupAMQP091 {
		return fmt.Errorf("unknown protocol %q", proto)
	}

	cfg := struct {
		URI              string   `json:"uri"`
		Exchange         string   `json:"exchange"`
		ExchangeType     string   `json:"exchangeType"`
		Queue            string   `json:"queue"`
		QueueDurable     bool     `json:"queueDurable"`
		BindingKeys      []string `json:"bindingKeys"`
		ConsumerTag      string   `json:"consumerTag"`
		PrefetchCount    int      `json:"prefetchCount"`
		AutoAck          bool     `json:"autoAck"`
		HeartbeatSeconds int      `json:"heartbeatSeconds"`
		ConnectionName   string   `json:"connectionName"`
		LocalAccount     string   `json:"localAccount"`
		RuleName         string   `json:"ruleName"`
		FilenameHeader   string   `json:"filenameHeader"`
		FilenameTemplate string   `json:"filenameTemplate"`
	}{}
	if err := utils.JSONConvert(conf, &cfg); err != nil {
		return err
	}
	if cfg.URI == "" || cfg.Queue == "" || cfg.LocalAccount == "" || cfg.RuleName == "" {
		return fmt.Errorf("invalid AMQP 0.9.1 server configuration")
	}

	return nil
}

func (backupAMQP091ConfigChecker) CheckClientConfig(proto string, conf map[string]any) error {
	if proto != backupAMQP091 {
		return fmt.Errorf("unknown protocol %q", proto)
	}

	cfg := struct {
		URI string `json:"uri"`
	}{}
	if err := utils.JSONConvert(conf, &cfg); err != nil {
		return err
	}
	if cfg.URI == "" {
		return fmt.Errorf("invalid AMQP 0.9.1 client configuration")
	}

	return nil
}

func (backupAMQP091ConfigChecker) CheckPartnerConfig(proto string, conf map[string]any) error {
	if proto != backupAMQP091 {
		return fmt.Errorf("unknown protocol %q", proto)
	}

	cfg := struct {
		Exchange string `json:"exchange"`
		Queue    string `json:"queue"`
	}{}
	if err := utils.JSONConvert(conf, &cfg); err != nil {
		return err
	}
	if cfg.Exchange == "" && cfg.Queue == "" {
		return fmt.Errorf("invalid AMQP 0.9.1 partner configuration")
	}

	return nil
}

func setBackupAMQP091ConfigChecker(t *testing.T) {
	t.Helper()

	oldChecker := model.ConfigChecker
	model.ConfigChecker = backupAMQP091ConfigChecker{}
	t.Cleanup(func() { model.ConfigChecker = oldChecker })
}

func assertConfigJSONEq(t *testing.T, expected, actual map[string]any) {
	t.Helper()

	expectedJSON, err := json.Marshal(expected)
	require.NoError(t, err)

	actualJSON, err := json.Marshal(actual)
	require.NoError(t, err)

	assert.JSONEq(t, string(expectedJSON), string(actualJSON))
}

func TestAMQP091ExportRoundTripSurfacesConfigs(t *testing.T) {
	setBackupAMQP091ConfigChecker(t)

	db := dbtest.TestDatabase(t)

	clientConfig := map[string]any{
		"uri":                "amqps://broker.example.net:5671/vh-client",
		"exchange":           "documents",
		"exchangeType":       "topic",
		"routingKeyTemplate": "gateway.${transferID}",
		"persistentMessages": true,
		"publisherConfirms":  true,
		"connectionName":     "gw-amqp091-client",
	}
	remoteConfig := map[string]any{
		"exchange":      "documents",
		"queue":         "partner.inbox",
		"queueDurable":  true,
		"bindingKeys":   []any{"partner.*"},
		"consumerTag":   "partner-consumer",
		"prefetchCount": 5,
	}
	localConfig := map[string]any{
		"uri":            "amqps://broker.example.net:5671/vh-server",
		"exchange":       "documents",
		"exchangeType":   "topic",
		"queue":          "gateway.inbox",
		"queueDurable":   true,
		"bindingKeys":    []any{"gateway.#"},
		"consumerTag":    "gateway-consumer",
		"prefetchCount":  8,
		"connectionName": "gw-amqp091-server",
		"localAccount":   "amqp-user",
		"ruleName":       "amqp-receive",
		"filenameHeader": "filename",
	}

	client := &model.Client{
		Name:         "amqp-client",
		Protocol:     backupAMQP091,
		LocalAddress: types.Addr("localhost", 3010),
		ProtoConfig:  clientConfig,
	}
	require.NoError(t, db.Insert(client).Run())

	remote := &model.RemoteAgent{
		Name:        "amqp-partner",
		Protocol:    backupAMQP091,
		Address:     types.Addr("localhost", 5671),
		ProtoConfig: remoteConfig,
	}
	require.NoError(t, db.Insert(remote).Run())
	require.NoError(t, db.Insert(&model.RemoteAccount{
		RemoteAgentID: remote.ID,
		Login:         "partner-user",
	}).Run())

	local := &model.LocalAgent{
		Name:        "amqp-server",
		Protocol:    backupAMQP091,
		Address:     types.Addr("localhost", 0),
		ProtoConfig: localConfig,
	}
	require.NoError(t, db.Insert(local).Run())
	require.NoError(t, db.Insert(&model.LocalAccount{
		LocalAgentID: local.ID,
		Login:        "amqp-user",
	}).Run())

	logger := logging.Discard()

	clients, err := exportClients(logger, db)
	require.NoError(t, err)
	require.Len(t, clients, 1)
	assert.Equal(t, backupAMQP091, clients[0].Protocol)
	assertConfigJSONEq(t, client.ProtoConfig, clients[0].ProtoConfig)

	remotes, err := exportRemotes(logger, db)
	require.NoError(t, err)
	require.Len(t, remotes, 1)
	assert.Equal(t, backupAMQP091, remotes[0].Protocol)
	assertConfigJSONEq(t, remote.ProtoConfig, remotes[0].Configuration)
	require.Len(t, remotes[0].Accounts, 1)
	assert.Equal(t, "partner-user", remotes[0].Accounts[0].Login)

	locals, err := exportLocals(logger, db)
	require.NoError(t, err)
	require.Len(t, locals, 1)
	assert.Equal(t, backupAMQP091, locals[0].Protocol)
	assertConfigJSONEq(t, local.ProtoConfig, locals[0].Configuration)
	require.Len(t, locals[0].Accounts, 1)
	assert.Equal(t, "amqp-user", locals[0].Accounts[0].Login)
}

func TestAMQP091ImportRoundTripValidatesRealConfigs(t *testing.T) {
	setBackupAMQP091ConfigChecker(t)

	db := dbtest.TestDatabase(t)

	clientSrc := backupfile.Client{
		Name:         "amqp-client",
		Protocol:     backupAMQP091,
		LocalAddress: "localhost:3010",
		ProtoConfig: map[string]any{
			"uri":                "amqps://broker.example.net:5671/vh-client",
			"exchange":           "documents",
			"exchangeType":       "topic",
			"routingKeyTemplate": "gateway.${transferID}",
			"persistentMessages": true,
		},
	}
	remoteSrc := backupfile.RemoteAgent{
		Name:     "amqp-partner",
		Protocol: backupAMQP091,
		Address:  "localhost:5671",
		Configuration: map[string]any{
			"exchange":     "documents",
			"queue":        "partner.inbox",
			"queueDurable": true,
			"bindingKeys":  []any{"partner.*"},
		},
		Accounts: []backupfile.RemoteAccount{{
			Login: "partner-user",
		}},
	}
	localSrc := backupfile.LocalAgent{
		Name:     "amqp-server",
		Protocol: backupAMQP091,
		Address:  "localhost:0",
		Configuration: map[string]any{
			"uri":          "amqps://broker.example.net:5671/vh-server",
			"queue":        "gateway.inbox",
			"queueDurable": true,
			"bindingKeys":  []any{"gateway.#"},
			"localAccount": "amqp-user",
			"ruleName":     "amqp-receive",
		},
		Accounts: []backupfile.LocalAccount{{
			Login:    "amqp-user",
			Password: "secret",
		}},
	}

	logger := logging.Discard()

	require.NoError(t, importClients(logger, db, []backupfile.Client{clientSrc}, false))
	require.NoError(t, importRemoteAgents(logger, db, []backupfile.RemoteAgent{remoteSrc}, false))
	require.NoError(t, importLocalAgents(logger, db, []backupfile.LocalAgent{localSrc}, false))

	var dbClient model.Client
	require.NoError(t, db.Get(&dbClient, "owner=? AND name=?", conf.GlobalConfig.GatewayName, clientSrc.Name).Run())
	assert.Equal(t, backupAMQP091, dbClient.Protocol)
	assertConfigJSONEq(t, clientSrc.ProtoConfig, dbClient.ProtoConfig)

	var dbRemote model.RemoteAgent
	require.NoError(t, db.Get(&dbRemote, "owner=? AND name=?", conf.GlobalConfig.GatewayName, remoteSrc.Name).Run())
	assert.Equal(t, backupAMQP091, dbRemote.Protocol)
	assertConfigJSONEq(t, remoteSrc.Configuration, dbRemote.ProtoConfig)

	var dbRemoteAccounts model.RemoteAccounts
	require.NoError(t, db.Select(&dbRemoteAccounts).Where("remote_agent_id=?", dbRemote.ID).Run())
	require.Len(t, dbRemoteAccounts, 1)
	assert.Equal(t, "partner-user", dbRemoteAccounts[0].Login)

	var dbLocal model.LocalAgent
	require.NoError(t, db.Get(&dbLocal, "owner=? AND name=?", conf.GlobalConfig.GatewayName, localSrc.Name).Run())
	assert.Equal(t, backupAMQP091, dbLocal.Protocol)
	assertConfigJSONEq(t, localSrc.Configuration, dbLocal.ProtoConfig)

	var dbLocalAccounts model.LocalAccounts
	require.NoError(t, db.Select(&dbLocalAccounts).Where("local_agent_id=?", dbLocal.ID).Run())
	require.Len(t, dbLocalAccounts, 1)
	assert.Equal(t, "amqp-user", dbLocalAccounts[0].Login)
}
