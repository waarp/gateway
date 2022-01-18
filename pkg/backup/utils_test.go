package backup

import (
	"path"

	"code.waarp.fr/lib/log"
	"github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/modeltest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	_ "code.waarp.fr/apps/gateway/gateway/pkg/tasks"
)

const testProtocol = "test_proto"

//nolint:gochecknoinits // init is used by design
func init() {
	modeltest.AddDummyProtoConfig(testProtocol)
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

func mkURL(elem ...string) *types.URL {
	full := path.Join(elem...)

	url, err := types.ParseURL(full)
	convey.So(err, convey.ShouldBeNil)

	return url
}

func mustAddr(addr string) types.Address {
	a, err := types.NewAddress(addr)
	convey.So(err, convey.ShouldBeNil)

	return *a
}
