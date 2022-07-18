package r66

import (
	"path"

	"code.waarp.fr/lib/r66"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/r66/internal"
)

//nolint:gochecknoglobals // these are test variables
var (
	servConf = &config.R66ServerProtoConfig{ServerLogin: "r66_login", ServerPassword: "sesame"}
	partConf = &config.R66PartnerProtoConfig{ServerLogin: "r66_login", ServerPassword: "sesame"}
)

func resetConnPool() {
	clientConns = internal.NewConnPool()
	Reset(clientConns.ForceClose)
}

func hash(pwd string) string {
	crypt := r66.CryptPass([]byte(pwd))
	h, err := bcrypt.GenerateFromPassword(crypt, bcrypt.MinCost)
	So(err, ShouldBeNil)

	return string(h)
}

func mkURL(elem ...string) *types.URL {
	full := path.Join(elem...)

	url, err := types.ParseURL(full)
	So(err, ShouldBeNil)

	return url
}
