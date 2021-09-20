package backup

import (
	"github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	_ "code.waarp.fr/apps/gateway/gateway/pkg/tasks"
)

//nolint:gochecknoglobals // global var is used by design
var discard *log.Logger

//nolint:gochecknoinits // init is used by design
func init() {
	logConf := conf.LogConfig{LogTo: "/dev/null"}
	_ = log.InitBackend(logConf)
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
