package gatewayd

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocolstest"
)

const testProtocol = "test_proto"

//nolint:gochecknoinits // init is used by design
func init() {
	protocols.Register(testProtocol, protocolstest.TestModule{})
}
