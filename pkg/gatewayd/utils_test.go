package gatewayd

import (
	"context"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/constructors"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/proto"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/state"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

const testProtocol = "test_proto"

//nolint:gochecknoinits // init is used by design
func init() {
	config.ProtoConfigs[testProtocol] = func() config.ProtoConfig {
		return new(testhelpers.TestProtoConfig)
	}

	constructors.ServiceConstructors[testProtocol] = newTestServ
}

type testServ struct{ state *state.State }

func newTestServ(*database.DB, *log.Logger) proto.Service {
	return &testServ{state: &state.State{}}
}

func (t *testServ) Start(*model.LocalAgent) error {
	t.state.Set(state.Running, "")

	return nil
}

func (t *testServ) Stop(context.Context) error {
	t.state.Set(state.Offline, "")

	return nil
}

func (t *testServ) State() *state.State                   { return t.state }
func (t *testServ) ManageTransfers() *service.TransferMap { return service.NewTransferMap() }
