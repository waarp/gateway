package testhelpers

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
)

// TestProtocol is the constant defining the name of the dummy protocol associated
// with the TestProtoConfig struct.
const TestProtocol = "test"

func init() {
	config.ProtoConfigs[TestProtocol] = func() config.ProtoConfig { return new(TestProtoConfig) }
}

// TestProtoConfig is a dummy implementation of config.ProtoConfig for test purposes.
type TestProtoConfig struct{}

// ValidServer is a dummy implementation of the server validation function.
// It does nothing, and never fails.
func (*TestProtoConfig) ValidServer() error { return nil }

// ValidPartner is a dummy implementation of the partner validation function.
// It does nothing, and never fails.
func (*TestProtoConfig) ValidPartner() error { return nil }
