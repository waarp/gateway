// Package protocolstest provides a full dummy implementation of a protocol
// module for test purposes.
package protocolstest

import (
	"slices"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/features"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
)

type TestModule struct{}

func (t TestModule) OptionalFeatures() []features.Feature {
	return slices.Collect(features.AllFeatures())
}
func (t TestModule) CanMakeTransfer(*model.TransferContext) error { return nil }
func (t TestModule) CheckServerConfig(map[string]any) error       { return nil }
func (t TestModule) CheckClientConfig(map[string]any) error       { return nil }
func (t TestModule) CheckPartnerConfig(map[string]any) error      { return nil }

func (t TestModule) NewServer(db *database.DB, server *model.LocalAgent) protocol.Server {
	return &TestServer{db: db, agent: server}
}

func (t TestModule) NewClient(db *database.DB, client *model.Client) protocol.Client {
	return &TestClient{db: db, agent: client}
}
