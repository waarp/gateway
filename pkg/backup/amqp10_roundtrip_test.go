package backup

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backupfile "code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const backupAMQP10 = "amqp10"

type backupAMQP10ConfigChecker struct{}

func (backupAMQP10ConfigChecker) IsValidProtocol(proto string) bool { return proto == backupAMQP10 }

func (backupAMQP10ConfigChecker) CheckServerConfig(proto string, conf map[string]any) error {
	if proto != backupAMQP10 {
		return fmt.Errorf("unknown protocol %q", proto)
	}

	cfg := struct {
		Endpoint         string `json:"endpoint"`
		SourceAddress    string `json:"sourceAddress"`
		ReceiverLinkName string `json:"receiverLinkName"`
		Credit           int    `json:"credit"`
		LocalAccount     string `json:"localAccount"`
		RuleName         string `json:"ruleName"`
		FilenameTemplate string `json:"filenameTemplate"`
	}{}
	if err := utils.JSONConvert(conf, &cfg); err != nil {
		return err
	}
	if cfg.Endpoint == "" || cfg.SourceAddress == "" || cfg.LocalAccount == "" || cfg.RuleName == "" {
		return fmt.Errorf("invalid AMQP 1.0 server configuration")
	}

	return nil
}

func (backupAMQP10ConfigChecker) CheckClientConfig(proto string, conf map[string]any) error {
	if proto != backupAMQP10 {
		return fmt.Errorf("unknown protocol %q", proto)
	}

	cfg := struct {
		Endpoint      string `json:"endpoint"`
		TargetAddress string `json:"targetAddress"`
	}{}
	if err := utils.JSONConvert(conf, &cfg); err != nil {
		return err
	}
	if cfg.Endpoint == "" || cfg.TargetAddress == "" {
		return fmt.Errorf("invalid AMQP 1.0 client configuration")
	}

	return nil
}

func (backupAMQP10ConfigChecker) CheckPartnerConfig(proto string, conf map[string]any) error {
	if proto != backupAMQP10 {
		return fmt.Errorf("unknown protocol %q", proto)
	}

	cfg := struct {
		SourceAddress string `json:"sourceAddress"`
	}{}
	if err := utils.JSONConvert(conf, &cfg); err != nil {
		return err
	}
	if cfg.SourceAddress == "" {
		return fmt.Errorf("invalid AMQP 1.0 partner configuration")
	}

	return nil
}

func setBackupAMQP10ConfigChecker(t *testing.T) {
	t.Helper()

	oldChecker := model.ConfigChecker
	model.ConfigChecker = backupAMQP10ConfigChecker{}
	t.Cleanup(func() { model.ConfigChecker = oldChecker })
}

func TestAMQP10ExportRoundTripSurfacesConfigs(t *testing.T) {
	setBackupAMQP10ConfigChecker(t)

	db := dbtest.TestDatabase(t)

	clientConfig := map[string]any{
		"endpoint":       "amqps://broker.example.net:5671",
		"targetAddress":  "gateway/out",
		"senderLinkName": "gateway-amqp10-out",
		"settlementMode": "mixed",
		"durable":        true,
		"connectionName": "gw-amqp10-client",
	}
	remoteConfig := map[string]any{
		"sourceAddress":    "gateway/in",
		"receiverLinkName": "gateway-amqp10-in",
		"credit":           10,
		"settlementMode":   "peek-lock",
	}
	localConfig := map[string]any{
		"endpoint":         "amqps://broker.example.net:5671",
		"sourceAddress":    "gateway/server-in",
		"receiverLinkName": "gateway-amqp10-server-in",
		"credit":           5,
		"localAccount":     "amqp-user",
		"ruleName":         "amqp10-receive",
		"filenameTemplate": "${messageID}",
	}

	client := &model.Client{
		Name:         "amqp10-client",
		Protocol:     backupAMQP10,
		LocalAddress: types.Addr("localhost", 3011),
		ProtoConfig:  clientConfig,
	}
	require.NoError(t, db.Insert(client).Run())

	remote := &model.RemoteAgent{
		Name:        "amqp10-partner",
		Protocol:    backupAMQP10,
		Address:     types.Addr("localhost", 5671),
		ProtoConfig: remoteConfig,
	}
	require.NoError(t, db.Insert(remote).Run())
	require.NoError(t, db.Insert(&model.RemoteAccount{
		RemoteAgentID: remote.ID,
		Login:         "partner-user",
	}).Run())

	local := &model.LocalAgent{
		Name:        "amqp10-server",
		Protocol:    backupAMQP10,
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
	assert.Equal(t, backupAMQP10, clients[0].Protocol)
	assertConfigJSONEq(t, client.ProtoConfig, clients[0].ProtoConfig)

	remotes, err := exportRemotes(logger, db)
	require.NoError(t, err)
	require.Len(t, remotes, 1)
	assert.Equal(t, backupAMQP10, remotes[0].Protocol)
	assertConfigJSONEq(t, remote.ProtoConfig, remotes[0].Configuration)
	require.Len(t, remotes[0].Accounts, 1)
	assert.Equal(t, "partner-user", remotes[0].Accounts[0].Login)

	locals, err := exportLocals(logger, db)
	require.NoError(t, err)
	require.Len(t, locals, 1)
	assert.Equal(t, backupAMQP10, locals[0].Protocol)
	assertConfigJSONEq(t, local.ProtoConfig, locals[0].Configuration)
	require.Len(t, locals[0].Accounts, 1)
	assert.Equal(t, "amqp-user", locals[0].Accounts[0].Login)
}

func TestAMQP10ImportRoundTripValidatesRealConfigs(t *testing.T) {
	setBackupAMQP10ConfigChecker(t)

	db := dbtest.TestDatabase(t)

	clientSrc := backupfile.Client{
		Name:         "amqp10-client",
		Protocol:     backupAMQP10,
		LocalAddress: "localhost:3011",
		ProtoConfig: map[string]any{
			"endpoint":       "amqps://broker.example.net:5671",
			"targetAddress":  "gateway/out",
			"senderLinkName": "gateway-amqp10-out",
		},
	}
	remoteSrc := backupfile.RemoteAgent{
		Name:     "amqp10-partner",
		Protocol: backupAMQP10,
		Address:  "localhost:5671",
		Configuration: map[string]any{
			"sourceAddress":    "gateway/in",
			"receiverLinkName": "gateway-amqp10-in",
			"credit":           20,
		},
		Accounts: []backupfile.RemoteAccount{{
			Login: "partner-user",
		}},
	}
	localSrc := backupfile.LocalAgent{
		Name:     "amqp10-server",
		Protocol: backupAMQP10,
		Address:  "localhost:0",
		Configuration: map[string]any{
			"endpoint":         "amqps://broker.example.net:5671",
			"sourceAddress":    "gateway/server-in",
			"receiverLinkName": "gateway-amqp10-server-in",
			"localAccount":     "amqp-user",
			"ruleName":         "amqp10-receive",
			"filenameTemplate": "${messageID}",
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

	var clients model.Clients
	require.NoError(t, db.Select(&clients).Run())
	require.Len(t, clients, 1)
	assert.Equal(t, backupAMQP10, clients[0].Protocol)
	assertConfigJSONEq(t, clientSrc.ProtoConfig, clients[0].ProtoConfig)

	var remotes model.RemoteAgents
	require.NoError(t, db.Select(&remotes).Run())
	require.Len(t, remotes, 1)
	assert.Equal(t, backupAMQP10, remotes[0].Protocol)
	assertConfigJSONEq(t, remoteSrc.Configuration, remotes[0].ProtoConfig)

	var locals model.LocalAgents
	require.NoError(t, db.Select(&locals).Run())
	require.Len(t, locals, 1)
	assert.Equal(t, backupAMQP10, locals[0].Protocol)
	assertConfigJSONEq(t, localSrc.Configuration, locals[0].ProtoConfig)
}

func TestAssertConfigJSONEqAMQP10(t *testing.T) {
	expected := map[string]any{"endpoint": "amqps://broker.example.net:5671", "credit": 5}
	actual := map[string]any{"credit": 5, "endpoint": "amqps://broker.example.net:5671"}

	expectedJSON, err := json.Marshal(expected)
	require.NoError(t, err)
	actualJSON, err := json.Marshal(actual)
	require.NoError(t, err)
	assert.JSONEq(t, string(expectedJSON), string(actualJSON))
}
