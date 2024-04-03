package r66

import (
	"testing"
	"time"

	"code.waarp.fr/lib/r66"
	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/fstest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestGetFileInfo(t *testing.T) {
	Convey("Given an R66 server", t, func(c C) {
		testFS := fstest.InitMemFS(c)
		logger := testhelpers.TestLogger(c, "test_r66_file_info")
		root := "memory:/r66_get_file_info"
		rootPath := mkURL(root)
		db := database.TestDatabase(c)
		conf.GlobalConfig.Paths.GatewayHome = root

		agent := &model.LocalAgent{
			Name: "r66_server", Protocol: "r66",
			RootDir: "r66_root", SendDir: "send",
			Address: types.Addr("localhost", 0),
			ProtoConfig: map[string]any{
				"serverLogin": "r66_server", "serverPassword": "foobar",
			},
		}
		So(db.Insert(agent).Run(), ShouldBeNil)

		account := &model.LocalAccount{
			LocalAgentID: agent.ID,
			Login:        "foo",
		}
		So(db.Insert(account).Run(), ShouldBeNil)

		accPswd := &model.Credential{
			LocalAccountID: utils.NewNullInt64(account.ID),
			Type:           auth.PasswordHash,
			Value:          "bar",
		}
		So(db.Insert(accPswd).Run(), ShouldBeNil)

		rule := &model.Rule{
			Name:   "send",
			IsSend: true,
			Path:   "send",
		}
		So(db.Insert(rule).Run(), ShouldBeNil)

		handle := sessionHandler{
			authHandler: &authHandler{
				service: &service{
					db:     db,
					logger: logger,
					agent:  agent,
				},
			},
			account: account,
		}

		Convey("Given a few files & directories", func() {
			dir := rootPath.JoinPath(agent.RootDir, agent.SendDir)
			subDir := dir.JoinPath("subDir")
			fooDir := subDir.JoinPath("fooDir")
			barDir := fooDir.JoinPath("barDir")

			foobar := subDir.JoinPath("foobar")
			toto := subDir.JoinPath("toto")
			tata := fooDir.JoinPath("tata")
			tutu := barDir.JoinPath("tutu")

			So(fs.MkdirAll(testFS, subDir), ShouldBeNil)
			So(fs.MkdirAll(testFS, fooDir), ShouldBeNil)
			So(fs.MkdirAll(testFS, barDir), ShouldBeNil)
			So(fs.WriteFullFile(testFS, foobar, []byte("foobar")), ShouldBeNil)
			So(fs.WriteFullFile(testFS, toto, []byte("toto")), ShouldBeNil)
			So(fs.WriteFullFile(testFS, tata, []byte("tata")), ShouldBeNil)
			So(fs.WriteFullFile(testFS, tutu, []byte("tutu")), ShouldBeNil)

			Convey("When calling the GetFileInfo function", func() {
				infos, err := handle.GetFileInfo(rule.Name, "subDir/foo*")
				So(err, ShouldBeNil)

				Convey("Then it should have returned the matching files", func() {
					So(infos, ShouldHaveLength, 2)
					So(infos[0].Name, ShouldEqual, "subDir/fooDir")
					So(infos[1].Name, ShouldEqual, "subDir/foobar")
				})
			})

			Convey("When calling the GetFileInfo function with an incorrect pattern", func() {
				_, err := handle.GetFileInfo(rule.Name, "barfoo")

				Convey("Then it should return an error", func() {
					So(err, ShouldBeError, &r66.Error{
						Code:   r66.FileNotFound,
						Detail: "no files found for the given pattern",
					})
				})
			})

			Convey("When calling the GetFileInfo function with an unknown rule", func() {
				_, err := handle.GetFileInfo("no_rule", "")

				Convey("Then it should return an error", func() {
					So(err, ShouldBeError, &r66.Error{
						Code:   r66.IncorrectCommand,
						Detail: "rule not found",
					})
				})
			})

			Convey("Given that the user is not allowed to use the given rule", func() {
				other := &model.LocalAccount{
					LocalAgentID: agent.ID,
					Login:        "other",
				}
				So(db.Insert(other).Run(), ShouldBeNil)

				otherPswd := &model.Credential{
					LocalAccountID: utils.NewNullInt64(other.ID),
					Type:           auth.PasswordHash,
					Value:          "other_pswd",
				}
				So(db.Insert(otherPswd).Run(), ShouldBeNil)

				accs := &model.RuleAccess{
					RuleID:         rule.ID,
					LocalAccountID: utils.NewNullInt64(other.ID),
				}
				So(db.Insert(accs).Run(), ShouldBeNil)

				Convey("When calling the GetFileInfo function", func() {
					_, err := handle.GetFileInfo(rule.Name, "")

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, &r66.Error{
							Code:   r66.IncorrectCommand,
							Detail: "you do not have the rights to use this transfer rule",
						})
					})
				})
			})
		})
	})
}

func TestGetTransferInfo(t *testing.T) {
	Convey("Given an R66 server", t, func(c C) {
		fstest.InitMemFS(c)
		logger := testhelpers.TestLogger(c, "test_r66_transfer_info")
		root := "memory:/r66_get_transfer_info"
		db := database.TestDatabase(c)
		conf.GlobalConfig.Paths.GatewayHome = root

		agent := &model.LocalAgent{
			Name: "r66_server", Protocol: "r66",
			RootDir: "r66_root", SendDir: "send",
			Address: types.Addr("localhost", 0),
			ProtoConfig: map[string]any{
				"serverLogin": "r66_server", "serverPassword": "foobar",
			},
		}
		So(db.Insert(agent).Run(), ShouldBeNil)

		account := &model.LocalAccount{
			LocalAgentID: agent.ID,
			Login:        "foo",
		}
		So(db.Insert(account).Run(), ShouldBeNil)

		accPswd := &model.Credential{
			LocalAccountID: utils.NewNullInt64(account.ID),
			Type:           auth.PasswordHash,
			Value:          "bar",
		}
		So(db.Insert(accPswd).Run(), ShouldBeNil)

		rule := &model.Rule{
			Name:     "snd",
			IsSend:   true,
			Path:     "snd",
			LocalDir: "snd_dir",
		}
		So(db.Insert(rule).Run(), ShouldBeNil)

		handle := sessionHandler{
			authHandler: &authHandler{
				service: &service{
					db:     db,
					logger: logger,
					agent:  agent,
				},
			},
			account: account,
		}

		Convey("Given a transfer on the R66 server", func() {
			trans := &model.Transfer{
				RemoteTransferID: "123",
				RuleID:           rule.ID,
				LocalAccountID:   utils.NewNullInt64(account.ID),
				SrcFilename:      "file.ex",
				Filesize:         100,
				Start:            time.Date(2021, 2, 1, 0, 0, 0, 0, time.Local),
				Status:           types.StatusRunning,
				Step:             types.StepData,
				Progress:         50,
				TaskNumber:       0,
			}
			So(db.Insert(trans).Run(), ShouldBeNil)

			tInfo := &model.TransferInfo{
				TransferID: utils.NewNullInt64(trans.ID),
				Name:       "key",
				Value:      `"val"`,
			}
			So(db.Insert(tInfo).Run(), ShouldBeNil)

			Convey("When calling the GetTransferInfo function", func() {
				info, err := handle.GetTransferInfo(123, false)
				So(err, ShouldBeNil)

				Convey("Then it should return the correct information", func() {
					So(info, ShouldResemble, &r66.TransferInfo{
						ID:        123,
						Client:    account.Login,
						Server:    agent.Name,
						File:      "file.ex",
						Rule:      rule.Name,
						IsRecv:    rule.IsSend,
						IsMd5:     false,
						BlockSize: 65536,
						Info:      `{"key":"val"}`,
						Start:     trans.Start,
						Stop:      time.Time{},
					})
				})
			})

			Convey("When calling GetTransferInfo with an unknown ID", func() {
				_, err := handle.GetTransferInfo(789, false)

				Convey("Then it should return an error", func() {
					So(err, ShouldBeError, &r66.Error{
						Code:   r66.IncorrectCommand,
						Detail: "transfer not found",
					})
				})
			})

			Convey("When calling GetTransferInfo for a client transfer", func() {
				_, err := handle.GetTransferInfo(123, true)

				Convey("Then it should return an error", func() {
					So(err, ShouldBeError, &r66.Error{
						Code:   r66.IncorrectCommand,
						Detail: "requesting info on client transfers is forbidden",
					})
				})
			})
		})

		Convey("Given a history entry on the R66 server", func() {
			hist := &model.HistoryEntry{
				ID:               1,
				RemoteTransferID: "123",
				IsServer:         true,
				IsSend:           true,
				Rule:             rule.Name,
				Account:          account.Login,
				Agent:            agent.Name,
				Protocol:         "r66",
				SrcFilename:      "file.ex",
				Filesize:         100,
				Start:            time.Date(2021, 2, 1, 0, 0, 0, 0, time.Local),
				Stop:             time.Date(2021, 2, 2, 0, 0, 0, 0, time.Local),
				Status:           types.StatusDone,
				Step:             types.StepNone,
				Progress:         100,
				TaskNumber:       0,
			}
			So(db.Insert(hist).Run(), ShouldBeNil)

			Convey("When calling the GetTransferInfo function", func() {
				info, err := handle.GetTransferInfo(123, false)
				So(err, ShouldBeNil)

				Convey("Then it should return the correct information", func() {
					So(info, ShouldResemble, &r66.TransferInfo{
						ID:        123,
						Client:    account.Login,
						Server:    agent.Name,
						File:      "file.ex",
						Rule:      rule.Name,
						RuleMode:  uint32(r66.ModeRecv),
						BlockSize: 0,
						Info:      "{}",
						Start:     hist.Start,
						Stop:      hist.Stop,
					})
				})
			})
		})
	})
}
