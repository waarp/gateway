package internal

import (
	"code.waarp.fr/lib/r66"
	"github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

type testAuthHandler func(*r66.Authent) (r66.SessionHandler, error)

func (t testAuthHandler) ValidAuth(auth *r66.Authent) (r66.SessionHandler, error) {
	return t(auth)
}

type testSessionHandler func(*r66.Request) (r66.TransferHandler, error)

func (t testSessionHandler) ValidRequest(request *r66.Request) (r66.TransferHandler, error) {
	return t(request)
}

func mkPath(full string) types.FSPath {
	fPath, err := types.ParsePath(full)
	convey.So(err, convey.ShouldBeNil)

	return *fPath
}
