package tasks

import (
	"github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

const testProtocol = "test_proto"

//nolint:gochecknoinits // init is used to ease the tests
func init() {
	config.ProtoConfigs[testProtocol] = func() config.ProtoConfig {
		return new(testhelpers.TestProtoConfig)
	}
}

func makeURL(str string) types.URL {
	url, err := types.ParseURL(str)
	convey.So(err, convey.ShouldBeNil)

	return *url
}
