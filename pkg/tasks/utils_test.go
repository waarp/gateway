package tasks

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/fstest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
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
}

func makeURL(str string) types.URL {
	return types.URL{Scheme: fstest.MemScheme, OmitHost: true, Path: str}
}
