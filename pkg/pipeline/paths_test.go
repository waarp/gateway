package pipeline

import (
	"fmt"
	"path"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestPathBuilder(t *testing.T) {
	Convey("Given a Gateway configuration", t, func(c C) {
		db := database.TestDatabase(c)

		db.Config.Paths.GatewayHome = "/path_builder"
		db.Config.Paths.DefaultInDir = "gwIn"
		db.Config.Paths.DefaultOutDir = "gwOut"
		db.Config.Paths.DefaultTmpDir = "gwTmp"

		server := &model.LocalAgent{
			Name:          "server",
			Protocol:      testProtocol,
			RootDir:       "serRoot",
			ReceiveDir:    "serIn",
			SendDir:       "serOut",
			TmpReceiveDir: "serTmp",
			Address:       types.Addr("localhost", 0),
		}
		So(db.Insert(server).Run(), ShouldBeNil)

		acc := &model.LocalAccount{
			LocalAgentID: server.ID,
			Login:        "toto",
		}
		So(db.Insert(acc).Run(), ShouldBeNil)

		send := &model.Rule{
			Name:           "SEND",
			IsSend:         true,
			Path:           "/path",
			LocalDir:       "sendLoc",
			RemoteDir:      "sendRem",
			TmpLocalRcvDir: "sendTmp",
		}
		So(db.Insert(send).Run(), ShouldBeNil)

		recv := &model.Rule{
			Name:           "RECEIVE",
			IsSend:         false,
			Path:           "/path",
			LocalDir:       "recvLoc",
			RemoteDir:      "recvRem",
			TmpLocalRcvDir: "recvTmp",
		}
		So(db.Insert(recv).Run(), ShouldBeNil)

		Convey("Given an incoming transfer", func(c C) {
			trans := &model.Transfer{
				RuleID:         recv.ID,
				LocalAccountID: acc.NullableID(),
				DestFilename:   "file.txt",
			}
			So(db.Insert(trans).Run(), ShouldBeNil)

			file := trans.DestFilename

			logger := testhelpers.TestLogger(c, "test_pipeline_path")
			transCtx, err := model.GetTransferContext(db, logger, trans)
			So(err, ShouldBeNil)

			type testCase struct {
				serRoot, ruleLoc, ruleTmp string
				expTmp                    string
			}

			gwRoot := db.Config.Paths.GatewayHome
			testCases := []testCase{
				{"", "", "", path.Join(gwRoot, "gwTmp", file)},
				{"serRoot", "", "", path.Join(gwRoot, "serRoot", "serTmp", file)},
				{"", "recvLoc", "", path.Join(gwRoot, "recvLoc", file)},
				{"", "", "recvTmp", path.Join(gwRoot, "recvTmp", file)},
				{"serRoot", "recvLoc", "", path.Join(gwRoot, "serRoot", "recvLoc", file)},
				{"serRoot", "", "recvTmp", path.Join(gwRoot, "serRoot", "recvTmp", file)},
				{"", "recvLoc", "recvTmp", path.Join(gwRoot, "recvTmp", file)},
				{"serRoot", "recvLoc", "recvTmp", path.Join(gwRoot, "serRoot", "recvTmp", file)},
			}

			for _, tc := range testCases {
				testCaseName := fmt.Sprintf(
					"Given the following path parameters: %q %q %q",
					tc.serRoot, tc.ruleLoc, tc.ruleTmp,
				)
				Convey(testCaseName, func() {
					transCtx.LocalAgent.RootDir = tc.serRoot
					if tc.serRoot != "" {
						transCtx.LocalAgent.ReceiveDir = "serIn"
						transCtx.LocalAgent.SendDir = "serOut"
						transCtx.LocalAgent.TmpReceiveDir = "serTmp"
					} else {
						transCtx.LocalAgent.ReceiveDir = ""
						transCtx.LocalAgent.SendDir = ""
						transCtx.LocalAgent.TmpReceiveDir = ""
					}

					transCtx.Rule.LocalDir = tc.ruleLoc
					transCtx.Rule.TmpLocalRcvDir = tc.ruleTmp

					Convey("When building the filepath", func() {
						pip := &Pipeline{TransCtx: transCtx}
						pip.setFilePaths()

						Convey("Then it should have built the expected tmp path", func() {
							So(transCtx.Transfer.LocalPath, ShouldEqual, tc.expTmp)
						})
					})
				})
			}
		})

		Convey("Given a client transfer not started yet", func(c C) {
			client := &model.Client{Name: "client", Protocol: testProtocol}
			So(db.Insert(client).Run(), ShouldBeNil)

			partner := &model.RemoteAgent{Name: "partner", Protocol: testProtocol, Address: types.Addr("localhost", 1)}
			So(db.Insert(partner).Run(), ShouldBeNil)

			account := &model.RemoteAccount{RemoteAgentID: partner.ID, Login: "titi"}
			So(db.Insert(account).Run(), ShouldBeNil)

			trans := &model.Transfer{
				RuleID:          send.ID,
				ClientID:        client.NullableID(),
				RemoteAccountID: account.NullableID(),
				SrcFilename:     "dir/file.txt",
			}
			So(db.Insert(trans).Run(), ShouldBeNil)

			logger := testhelpers.TestLogger(c, "test_pipeline_path")
			transCtx, err := model.GetTransferContext(db, logger, trans)
			So(err, ShouldBeNil)

			Convey("When computing its expected paths", func() {
				local, remote, err := ExpectedPaths(transCtx)
				So(err, ShouldBeNil)

				Convey("Then they are the paths the pipeline sets at the start", func() {
					So(trans.LocalPath, ShouldBeEmpty)
					So(trans.RemotePath, ShouldBeEmpty)

					pip := &Pipeline{TransCtx: transCtx}
					So(pip.setFilePaths(), ShouldBeNil)
					So(local, ShouldEqual, trans.LocalPath)
					So(remote, ShouldEqual, trans.RemotePath)
					So(local, ShouldEqual, path.Join(db.Config.Paths.GatewayHome, "sendLoc", "dir/file.txt"))
					So(remote, ShouldEqual, "sendRem/dir/file.txt")
				})
			})
		})

		Convey("Given an outgoing transfer", func(c C) {
			trans := &model.Transfer{
				RuleID:         send.ID,
				LocalAccountID: acc.NullableID(),
				SrcFilename:    "file.txt",
			}
			So(db.Insert(trans).Run(), ShouldBeNil)

			file := trans.SrcFilename

			logger := testhelpers.TestLogger(c, "test_pipeline_path")
			transCtx, err := model.GetTransferContext(db, logger, trans)
			So(err, ShouldBeNil)

			type testCase struct {
				serRoot, ruleLoc string
				expFinal         string
			}

			gwRoot := db.Config.Paths.GatewayHome
			testCases := []testCase{
				{"", "", path.Join(gwRoot, "gwOut", file)},
				{"serRoot", "", path.Join(gwRoot, "serRoot", "serOut", file)},
				{"", "sendLoc", path.Join(gwRoot, "sendLoc", file)},
				{"serRoot", "sendLoc", path.Join(gwRoot, "serRoot", "sendLoc", file)},
			}

			for _, tc := range testCases {
				testCaseName := fmt.Sprintf(
					"Given the following path parameters: %q %q",
					tc.serRoot, tc.ruleLoc,
				)
				Convey(testCaseName, func() {
					transCtx.LocalAgent.RootDir = tc.serRoot
					if tc.serRoot != "" {
						transCtx.LocalAgent.ReceiveDir = "serIn"
						transCtx.LocalAgent.SendDir = "serOut"
						transCtx.LocalAgent.TmpReceiveDir = "serTmp"
					} else {
						transCtx.LocalAgent.ReceiveDir = ""
						transCtx.LocalAgent.SendDir = ""
						transCtx.LocalAgent.TmpReceiveDir = ""
					}

					transCtx.Rule.LocalDir = tc.ruleLoc

					Convey("When building the filepath", func() {
						pip := &Pipeline{TransCtx: transCtx}
						pip.setFilePaths()

						Convey("Then it should have built the expected out path", func() {
							So(transCtx.Transfer.LocalPath, ShouldEqual, tc.expFinal)
						})
					})
				})
			}
		})
	})
}
