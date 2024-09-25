package tasks

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/fstest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/modeltest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

const testProtocol = "test_proto"

//nolint:gochecknoinits // init is used to ease the tests
func init() {
	modeltest.AddDummyProtoConfig(testProtocol)
}

func makePath(str string) types.FSPath {
	return types.FSPath{Backend: fstest.MemBackend, Path: str}
}
