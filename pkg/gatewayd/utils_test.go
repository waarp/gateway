package gatewayd

import (
	"context"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/service"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

const testProtocol = "test_proto"

//nolint:gochecknoinits // init is used by design
func init() {
	config.ProtoConfigs[testProtocol] = func() config.ProtoConfig {
		return new(testhelpers.TestProtoConfig)
	}

	ServiceConstructors[testProtocol] = newTestServ
}

type testServ struct{ state *service.State }

func newTestServ(*database.DB, *model.LocalAgent, *log.Logger) service.ProtoService {
	return &testServ{state: &service.State{}}
}

func (t *testServ) Start() error {
	t.state.Set(service.Running, "")

	return nil
}

func (t *testServ) Stop(context.Context) error {
	t.state.Set(service.Offline, "")

	return nil
}

func (t *testServ) State() *service.State                 { return t.state }
func (t *testServ) ManageTransfers() *service.TransferMap { return service.NewTransferMap() }
