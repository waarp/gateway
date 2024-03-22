package model

import (
	"github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

const (
	testProtocol        = "test_proto"
	testProtocolInvalid = "test_proto_invalid"
)

var testLocalPath = "file:/test/local/file"

//nolint:gochecknoglobals // global var is simpler here
var testConfigMaker = &config.Constructor{
	Server:  func() config.ServerProtoConfig { return new(testhelpers.TestProtoConfig) },
	Partner: func() config.PartnerProtoConfig { return new(testhelpers.TestProtoConfig) },
	Client:  func() config.ClientProtoConfig { return new(testhelpers.TestProtoConfig) },
}

//nolint:gochecknoinits // init is used to ease the tests
func init() {
	config.ProtoConfigs[testProtocol] = testConfigMaker
	config.ProtoConfigs[testProtocolInvalid] = &config.Constructor{
		Server:  func() config.ServerProtoConfig { return new(testhelpers.TestProtoConfigFail) },
		Partner: func() config.PartnerProtoConfig { return new(testhelpers.TestProtoConfigFail) },
		Client:  func() config.ClientProtoConfig { return new(testhelpers.TestProtoConfigFail) },
	}
}

func hash(pwd string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.MinCost)
	convey.So(err, convey.ShouldBeNil)

	return string(h)
}

func mkURL(str string) types.URL {
	url, err := types.ParseURL(str)
	convey.So(err, convey.ShouldBeNil)

	return *url
}
