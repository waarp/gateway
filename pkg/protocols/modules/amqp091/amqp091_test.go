package amqp091

import (
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

func TestClientConfigValidatesAndNormalizes(t *testing.T) {
	conf := defaultClientConfig()
	conf.URI = "amqps://broker.example.net:5671"
	conf.ExchangeType = amqp.ExchangeTopic
	conf.RoutingKeyTemplate = "gateway.${transferID}"

	require.NoError(t, conf.ValidClient())
	assert.Equal(t, defaultHeartbeatSeconds, conf.HeartbeatSeconds)
	assert.Equal(t, defaultConsumeTimeout, conf.ConsumeTimeout)
}

func TestClientConfigRejectsInvalidURI(t *testing.T) {
	conf := defaultClientConfig()
	conf.URI = "https://broker.example.net"

	err := conf.ValidClient()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "amqp or amqps")
}

func TestPartnerConfigRequiresQueueOrExchange(t *testing.T) {
	conf := defaultPartnerConfig()

	err := conf.ValidPartner()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exchange or a queue")
}

func TestServerConfigRequiresQueue(t *testing.T) {
	conf := defaultServerConfig()
	conf.URI = "amqp://broker.example.net:5672"

	err := conf.ValidServer()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server queue")
}

func TestRenderRoutingKey(t *testing.T) {
	pip := &pipeline.Pipeline{
		Logger: logging.NewLogger("amqp091-test"),
		TransCtx: &model.TransferContext{
			Transfer: &model.Transfer{
				RemoteTransferID: "rt-123",
				SrcFilename:      "src.txt",
				DestFilename:     "dst.txt",
			},
			Rule: &model.Rule{Name: "rule-a"},
		},
	}

	got := renderRoutingKey("gateway.${ruleName}.${srcFilename}.${transferID}", pip)
	assert.Equal(t, "gateway.rule-a.src.txt.rt-123", got)
}
