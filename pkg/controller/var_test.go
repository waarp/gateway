package controller

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

const testProtocol = "test_proto"

//nolint:gochecknoinits // init is used to ease the tests
func init() {
	_ = log.InitBackend("DEBUG", "stdout", "")

	config.ProtoConfigs[testProtocol] = func() config.ProtoConfig {
		return new(testhelpers.TestProtoConfig)
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
