package model

import (
	"github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

const testProtocol = "test_proto"

//nolint:gochecknoinits // init is used to ease the tests
func init() {
	_ = log.InitBackend("DEBUG", "stdout", "")

	config.ProtoConfigs[testProtocol] = func() config.ProtoConfig {
		return new(testhelpers.TestProtoConfig)
	}
}

func hash(pwd string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.MinCost)
	convey.So(err, convey.ShouldBeNil)

	return string(h)
}
