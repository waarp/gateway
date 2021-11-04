package rest

import (
	"github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
)

const (
	testProto1 = "rest_test_1"
	testProto2 = "rest_test_2"
)

//nolint:gochecknoinits // init is used by design
func init() {
	config.ProtoConfigs[testProto1] = func() config.ProtoConfig { return new(TestProtoConfig) }
	config.ProtoConfigs[testProto2] = func() config.ProtoConfig { return new(TestProtoConfig) }

	_ = log.InitBackend("DEBUG", "stdout", "")
}

type TestProtoConfig struct {
	Key string `json:"key,omitempty"`
}

func (*TestProtoConfig) ValidServer() error  { return nil }
func (*TestProtoConfig) ValidPartner() error { return nil }

func hash(pwd string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.MinCost)
	convey.So(err, convey.ShouldBeNil)

	return string(h)
}
