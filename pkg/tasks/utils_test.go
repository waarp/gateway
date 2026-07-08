package tasks

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/modeltest"
)

const testProtocol = "test_proto"

//nolint:gochecknoinits // init is used to ease the tests
func init() {
	modeltest.AddDummyProtoConfig(testProtocol)

	fs.FilePerms = 0o644
	fs.DirPerms = 0o755
}
