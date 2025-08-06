package tasks

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/modeltest"
)

const testProtocol = "test_proto"

//nolint:gochecknoinits // init is used to ease the tests
func init() {
	modeltest.AddDummyProtoConfig(testProtocol)

	conf.GlobalConfig.Paths.FilePerms = 0o644
	conf.GlobalConfig.Paths.DirPerms = 0o755
}
