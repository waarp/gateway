// Package protocolstest provides a full dummy implementation of a protocol
// module for test purposes.
package protocolstest

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
)

type TestModule struct{}

func (TestModule) NewServer(*database.DB, *model.LocalAgent) protocol.Server {
	return &TestService{}
}

func (TestModule) NewClient(*database.DB, *model.Client) protocol.Client {
	return &TestService{}
}

func (TestModule) MakeServerConfig() protocol.ServerConfig   { return &TestConfig{} }
func (TestModule) MakeClientConfig() protocol.ClientConfig   { return &TestConfig{} }
func (TestModule) MakePartnerConfig() protocol.PartnerConfig { return &TestConfig{} }
