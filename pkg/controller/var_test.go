package controller

import (
	"path"

	"github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

const testProtocol = "test_proto"

//nolint:gochecknoinits // init is used to ease the tests
func init() {
	config.ProtoConfigs[testProtocol] = &config.ConfigMaker{
		Server:  func() config.ServerProtoConfig { return new(testhelpers.TestProtoConfig) },
		Partner: func() config.PartnerProtoConfig { return new(testhelpers.TestProtoConfig) },
		Client:  func() config.ClientProtoConfig { return new(testhelpers.TestProtoConfig) },
	}
	pipeline.ClientConstructors[testProtocol] = newAllSuccess
}

type allSuccess struct{}

func newAllSuccess(*pipeline.Pipeline) (pipeline.Client, *types.TransferError) {
	return &allSuccess{}, nil
}
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
