package r66

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"code.waarp.fr/lib/r66"
	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

//nolint:gochecknoinits // init is used by design
func init() {
	_ = log.InitBackend("DEBUG", "stdout", "")
}

func TestGetFileInfo(t *testing.T) {
	Convey("Given an R66 server", t, func(c C) {
		root := testhelpers.TempDir(c, "r66_get_file_info")
		db := database.TestDatabase(c, "ERROR")
		conf.GlobalConfig.Paths.GatewayHome = root

		protoConf, err := json.Marshal(config.R66ProtoConfig{
			ServerLogin: "r66_server", ServerPassword: "foobar",
		})
		So(err, ShouldBeNil)

		agent := &model.LocalAgent{
			Name:        "r66_server",
			Protocol:    "r66",
			RootDir:     "r66_root",
			SendDir:     "send",
			Address:     "localhost:6666",
			ProtoConfig: protoConf,
		}
		So(db.Insert(agent).Run(), ShouldBeNil)

		account := &model.LocalAccount{
			LocalAgentID: agent.ID,
			Login:        "foo",
			PasswordHash: hash("bar"),
		}
		So(db.Insert(account).Run(), ShouldBeNil)

		rule := &model.Rule{
			Name:   "send",
			IsSend: true,
			Path:   "send",
		}
		So(db.Insert(rule).Run(), ShouldBeNil)

		handle := sessionHandler{
			authHandler: &authHandler{
				Service: &Service{
					db:     db,
					logger: log.NewLogger("r66"),
					agent:  agent,
				},
			},
			account: account,
		}

		Convey("Given a few files & directories", func() {
			dir := filepath.Join(root, agent.RootDir, agent.SendDir)
			subDir := filepath.Join(dir, "subDir")
			fooDir := filepath.Join(subDir, "fooDir")
			barDir := filepath.Join(fooDir, "barDir")

			foobar := filepath.Join(subDir, "foobar")
			toto := filepath.Join(subDir, "toto")
			tata := filepath.Join(fooDir, "tata")
			tutu := filepath.Join(barDir, "tutu")

			So(os.MkdirAll(subDir, 0o700), ShouldBeNil)
			So(os.MkdirAll(fooDir, 0o700), ShouldBeNil)
			So(os.MkdirAll(barDir, 0o700), ShouldBeNil)
			So(os.WriteFile(foobar, []byte("foobar"), 0o600), ShouldBeNil)
			So(os.WriteFile(toto, []byte("toto"), 0o600), ShouldBeNil)
			So(os.WriteFile(tata, []byte("tata"), 0o600), ShouldBeNil)
			So(os.WriteFile(tutu, []byte("tutu"), 0o600), ShouldBeNil)

			Convey("When calling the GetFileInfo function", func() {
				infos, err := handle.GetFileInfo(rule.Name, "subDir/foo*")
				So(err, ShouldBeNil)

				Convey("Then it should have returned the matching files", func() {
					So(infos, ShouldHaveLength, 3)
					So(infos[0].Name, ShouldEqual, "subDir/fooDir/barDir")
					So(infos[1].Name, ShouldEqual, "subDir/fooDir/tata")
					So(infos[2].Name, ShouldEqual, "subDir/foobar")
				})
			})

			Convey("When calling the GetFileInfo function with an incorrect pattern", func() {
				_, err := handle.GetFileInfo(rule.Name, "barfoo")
				So(err, ShouldBeError, &r66.Error{
					Code:   r66.FileNotFound,
					Detail: "no files found for the given pattern",
				})
			})

			Convey("When calling the GetFileInfo function with an unknown rule", func() {
				_, err := handle.GetFileInfo("no_rule", "")
				So(err, ShouldBeError, &r66.Error{
					Code:   r66.IncorrectCommand,
					Detail: "rule not found",
				})
			})
		})
	})
}
