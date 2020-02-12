package rest

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
)

func init() {
	config.ProtoConfigs["test"] = func() config.ProtoConfig { return new(TestProtoConfig) }
}

var logConf = conf.LogConfig{
	Level: "DEBUG",
	LogTo: "stdout",
}

type TestProtoConfig struct{}

func (*TestProtoConfig) ValidServer() error { return nil }
func (*TestProtoConfig) ValidClient() error { return nil }
