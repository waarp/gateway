package r66

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"code.waarp.fr/lib/r66"
	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

func TestGetFileInfo(t *testing.T) {
	Convey("Given an R66 server", t, func(c C) {
		logger := testhelpers.TestLogger(c, "test_r66_file_info")
		root := testhelpers.TempDir(c, "r66_get_file_info")
		db := database.TestDatabase(c)
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
					db:      db,
					logger:  logger,
					agentID: agent.ID,
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
					PasswordHash: hash("other_pswd"),
				}
				So(db.Insert(other).Run(), ShouldBeNil)

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
		logger := testhelpers.TestLogger(c, "test_r66_transfer_info")
		root := testhelpers.TempDir(c, "r66_get_transfer_info")
		db := database.TestDatabase(c)
		conf.GlobalConfig.Paths.GatewayHome = root

		protoConfig := config.R66ProtoConfig{
			ServerLogin: "r66_server", ServerPassword: "foobar",
		}

		jsonProtoConf, err := json.Marshal(protoConfig)
		So(err, ShouldBeNil)

		agent := &model.LocalAgent{
			Name:        "r66_server",
			Protocol:    "r66",
			RootDir:     "r66_root",
			SendDir:     "send",
			Address:     "localhost:6666",
			ProtoConfig: jsonProtoConf,
		}
		So(db.Insert(agent).Run(), ShouldBeNil)

		account := &model.LocalAccount{
			LocalAgentID: agent.ID,
			Login:        "foo",
			PasswordHash: hash("bar"),
		}
		So(db.Insert(account).Run(), ShouldBeNil)

		rule := &model.Rule{
			Name:     "snd",
			IsSend:   true,
			Path:     "snd",
			LocalDir: "snd_dir",
		}
		So(db.Insert(rule).Run(), ShouldBeNil)

		handle := sessionHandler{
			authHandler: &authHandler{
				Service: &Service{
					db:      db,
					logger:  logger,
					agentID: agent.ID,
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
						IsMd5:     protoConfig.CheckBlockHash,
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
