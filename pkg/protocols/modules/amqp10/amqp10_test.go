package amqp10

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

func TestClientConfigValidatesAndNormalizes(t *testing.T) {
	conf := defaultClientConfig()
	conf.Endpoint = "amqps://broker.example.net:5671"
	conf.TargetAddress = "gateway/out"

	require.NoError(t, conf.ValidClient())
	assert.Equal(t, defaultIdleTimeoutSeconds, conf.IdleTimeoutSeconds)
	assert.Equal(t, defaultConsumeTimeout, conf.ConsumeTimeout)
	assert.Equal(t, 1, conf.MaxInFlight)
}

func TestClientConfigRejectsInvalidEndpoint(t *testing.T) {
	conf := defaultClientConfig()
	conf.Endpoint = "https://broker.example.net:443"
	conf.TargetAddress = "gateway/out"

	err := conf.ValidClient()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "amqp or amqps")
}

func TestPartnerConfigRequiresSourceAddress(t *testing.T) {
	conf := defaultPartnerConfig()

	err := conf.ValidPartner()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "source address")
}

func TestServerConfigRequiresSourceAddress(t *testing.T) {
	conf := defaultServerConfig()
	conf.Endpoint = "amqp://broker.example.net:5672"
	conf.LocalAccount = "inbox"
	conf.RuleName = "amqp10-in"

	err := conf.ValidServer()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sourceAddress")
}

func TestConnectionName(t *testing.T) {
	assert.Equal(t, "waarp-amqp10-rt-123", connectionName("", "rt-123"))
}

func TestStringifyAMQPValue(t *testing.T) {
	assert.Equal(t, "42", stringifyAMQPValue(42))
	assert.Equal(t, "", stringifyAMQPValue(nil))
}

func TestClientInitTransferRejectsInvalidPartnerConfig(t *testing.T) {
	cli := newClient(&model.Client{
		Name: "amqp10-client",
		ProtoConfig: map[string]any{
			"endpoint":      "amqps://broker.example.net:5671",
			"targetAddress": "gateway/out",
		},
	})
	require.NoError(t, cli.Start())

	pip := &pipeline.Pipeline{
		TransCtx: &model.TransferContext{
			RemoteAgent: &model.RemoteAgent{ProtoConfig: map[string]any{}},
			Rule:        &model.Rule{IsSend: false},
		},
	}

	tx, perr := cli.InitTransfer(pip)
	require.Nil(t, tx)
	require.NotNil(t, perr)
	assert.Contains(t, perr.Error(), "invalid AMQP 1.0 partner config")
}
