// Package protocolstest provides a full dummy implementation of a protocol
// module for test purposes.
package protocolstest

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
)

type TestModule struct{}

func (t TestModule) CanMakeTransfer(*model.TransferContext) error { return nil }
func (t TestModule) CheckServerConfig(map[string]any) error       { return nil }
func (t TestModule) CheckClientConfig(map[string]any) error       { return nil }
func (t TestModule) CheckPartnerConfig(map[string]any) error      { return nil }

func (t TestModule) NewServer(*database.DB, *model.LocalAgent) protocol.Server {
	return &TestService{}
}

func (t TestModule) NewClient(*database.DB, *model.Client) protocol.Client {
	return &TestService{}
}
