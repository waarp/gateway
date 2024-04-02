package r66

import (
	"path"

	"code.waarp.fr/lib/r66"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
)

//nolint:gochecknoglobals // these are test variables
var (
	cliConf  = &clientConfig{}
	servConf = &serverConfig{ServerLogin: "r66_login", ServerPassword: "sesame"}
	partConf = &partnerConfig{ServerLogin: "r66_login", ServerPassword: "sesame"}
)

func init() {
	pipelinetest.Protocols[R66] = pipelinetest.ProtoFeatures{
		MakeClient: func(db *database.DB, cli *model.Client) services.Client {
			return &client{db: db, cli: cli, disableConnGrace: true}
		},
		MakeServer:        Module{}.NewServer,
		MakeServerConfig:  Module{}.MakeServerConfig,
		MakeClientConfig:  Module{}.MakeClientConfig,
		MakePartnerConfig: Module{}.MakePartnerConfig,
		TransID:           true,
		RuleName:          true,
		Size:              true,
	}
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
