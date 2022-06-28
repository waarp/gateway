package backup

import (
	"code.waarp.fr/lib/log"
	"github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	_ "code.waarp.fr/apps/gateway/gateway/pkg/tasks"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

const testProtocol = "test_proto"

//nolint:gochecknoinits // init is used by design
func init() {
	config.ProtoConfigs[testProtocol] = func() config.ProtoConfig {
		return new(testhelpers.TestProtoConfig)
	}
}

func discard() *log.Logger {
	back, err := log.NewBackend(log.LevelTrace, log.Discard, "", "")
	convey.So(err, convey.ShouldBeNil)

	return back.NewLogger("discard")
}

func hash(pwd string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.MinCost)
	convey.So(err, convey.ShouldBeNil)

	return string(h)
}
