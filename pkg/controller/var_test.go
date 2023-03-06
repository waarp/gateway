package controller

import (
	"context"
	"path"

	"github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/state"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

const testProtocol = "test_proto"

//nolint:gochecknoinits // init is used to ease the tests
func init() {
	config.ProtoConfigs[testProtocol] = &config.Constructor{
		Server:  func() config.ServerProtoConfig { return new(testhelpers.TestProtoConfig) },
		Partner: func() config.PartnerProtoConfig { return new(testhelpers.TestProtoConfig) },
		Client:  func() config.ClientProtoConfig { return new(testhelpers.TestProtoConfig) },
	}
}

type allSuccess struct{ state state.State }

func newAllSuccess() *allSuccess {
	cli := &allSuccess{}
	cli.state.Set(state.Running, "")

	return cli
}

func (a *allSuccess) Start() error        { return nil }
func (a *allSuccess) State() *state.State { return &a.state }

func (a *allSuccess) ManageTransfers() *service.TransferMap { return &service.TransferMap{} }
func (a *allSuccess) InitTransfer(*pipeline.Pipeline) (pipeline.TransferClient, *types.TransferError) {
	return a, nil
}
func (a *allSuccess) Stop(context.Context) error                    { return nil }
func (a *allSuccess) Request() *types.TransferError                 { return nil }
func (a *allSuccess) Data(pipeline.DataStream) *types.TransferError { return nil }
func (a *allSuccess) EndTransfer() *types.TransferError             { return nil }
func (a *allSuccess) SendError(*types.TransferError)                {}

func mkURL(elem ...string) *types.URL {
	full := path.Join(elem...)

	url, err := types.ParseURL(full)
	convey.So(err, convey.ShouldBeNil)

	return url
}
