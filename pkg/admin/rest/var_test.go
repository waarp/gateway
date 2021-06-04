package rest

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"
)

func init() {
	config.ProtoConfigs["test"] = func() config.ProtoConfig { return new(TestProtoConfig) }
	config.ProtoConfigs["test2"] = func() config.ProtoConfig { return new(TestProtoConfig) }

	_ = log.InitBackend("DEBUG", "stdout", "")
}

type TestProtoConfig struct {
	Key string `json:"key,omitempty"`
}

func (*TestProtoConfig) ValidServer() error  { return nil }
func (*TestProtoConfig) ValidPartner() error { return nil }

func hash(pwd string) []byte {
	h, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.MinCost)
	convey.So(err, convey.ShouldBeNil)
	return h
}
