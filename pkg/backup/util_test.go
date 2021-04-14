package backup

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	_ "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tasks"
)

var discard *log.Logger

func init() {
	logConf := conf.LogConfig{LogTo: "/dev/null"}
	_ = log.InitBackend(logConf)
	discard = log.NewLogger("discard")

	config.ProtoConfigs["test"] = func() config.ProtoConfig { return new(TestProtoConfig) }
}

type TestProtoConfig struct{}

func (*TestProtoConfig) ValidServer() error  { return nil }
func (*TestProtoConfig) ValidPartner() error { return nil }
func (*TestProtoConfig) CertRequired() bool  { return false }
