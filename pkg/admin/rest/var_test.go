package rest

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
)

func init() {
	config.ProtoConfigs["test"] = func() config.ProtoConfig { return new(TestProtoConfig) }
	config.ProtoConfigs["test2"] = func() config.ProtoConfig { return new(TestProtoConfig) }

	_ = log.InitBackend("DEBUG", "stdout", "")
}

type TestProtoConfig struct {
	Key string `json:"key,omitempty"`
}

func (*TestProtoConfig) ValidServer() error  { return nil }
func (*TestProtoConfig) ValidPartner() error { return nil }
func (*TestProtoConfig) CertRequired() bool  { return false }
