package backup

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	_ "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tasks"
	"github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"
)

var discard *log.Logger

func init() {
	_ = log.InitBackend("CRITICAL", "/dev/null", "")
	discard = log.NewLogger("discard")

	config.ProtoConfigs["test"] = func() config.ProtoConfig { return new(TestProtoConfig) }
}

type TestProtoConfig struct{}

func (*TestProtoConfig) ValidServer() error  { return nil }
func (*TestProtoConfig) ValidPartner() error { return nil }
func (*TestProtoConfig) CertRequired() bool  { return false }

func hash(pwd string) []byte {
	h, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.MinCost)
	convey.So(err, convey.ShouldBeNil)
	return h
}
