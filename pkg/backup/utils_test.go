package backup

import (
	"runtime"

	"code.waarp.fr/lib/log"
	"github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/modeltest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66"
	_ "code.waarp.fr/apps/gateway/gateway/pkg/tasks"
)

const testProtocol = "test_proto"

//nolint:gochecknoinits // init is used by design
func init() {
	modeltest.AddDummyProtoConfig(testProtocol)
	modeltest.AddDummyProtoConfig(r66.R66TLS)
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

func localPath(full string) types.FSPath {
	if runtime.GOOS == "windows" {
		full = "C:" + full
	}

	fPath, err := types.ParsePath(full)
	convey.So(err, convey.ShouldBeNil)

	return *fPath
}

func mustAddr(addr string) types.Address {
	a, err := types.NewAddress(addr)
	convey.So(err, convey.ShouldBeNil)

	return *a
}
