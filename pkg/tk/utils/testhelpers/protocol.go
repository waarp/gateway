package testhelpers

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
)

const TestProtocol = "test"

func init() {
	config.ProtoConfigs[TestProtocol] = func() config.ProtoConfig { return new(TestProtoConfig) }
}

type TestProtoConfig struct{}

func (*TestProtoConfig) ValidServer() error  { return nil }
func (*TestProtoConfig) ValidPartner() error { return nil }
