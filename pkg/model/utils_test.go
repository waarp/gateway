package model

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
)

func init() {
	logConf := conf.LogConfig{
		Level: "CRITICAL",
		LogTo: "stdout",
	}
	_ = log.InitBackend(logConf)

	config.ProtoConfigs["dummy"] = func() config.ProtoConfig { return new(TestProtoConfig) }
}

type TestProtoConfig struct{}

func (*TestProtoConfig) ValidServer() error  { return nil }
func (*TestProtoConfig) ValidPartner() error { return nil }
func (*TestProtoConfig) CertRequired() bool  { return false }
