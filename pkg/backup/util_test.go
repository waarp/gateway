package backup

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	_ "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tasks"
	"github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"
)

var discard *log.Logger

func init() {
	_ = log.InitBackend("CRITICAL", "/dev/null", "")
	discard = log.NewLogger("discard")
}

func hash(pwd string) []byte {
	h, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.MinCost)
	convey.So(err, convey.ShouldBeNil)
	return h
}
