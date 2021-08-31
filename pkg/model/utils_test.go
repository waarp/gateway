package model

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"
)

const dummyProto = "dummy"

func init() {
	_ = log.InitBackend("DEBUG", "stdout", "")

	config.ProtoConfigs[dummyProto] = func() config.ProtoConfig { return new(TestProtoConfig) }
}

type TestProtoConfig struct{}

func (*TestProtoConfig) ValidServer() error  { return nil }
func (*TestProtoConfig) ValidPartner() error { return nil }

func hash(pwd string) []byte {
	h, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.MinCost)
	convey.So(err, convey.ShouldBeNil)
	return h
}
