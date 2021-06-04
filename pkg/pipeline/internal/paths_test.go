package internal

import (
	"encoding/json"
	"fmt"
	. "path/filepath"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"
	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	_ = log.InitBackend("DEBUG", "stdout", "")
}

func hash(pwd string) []byte {
	h, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.MinCost)
	So(err, ShouldBeNil)
	return h
}

func TestPathBuilder(t *testing.T) {
	logger := log.NewLogger("test_path_builder")

	Convey("Given a Gateway configuration", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		db.Conf.Paths.GatewayHome = testhelpers.TempDir(c, "path_builder")
		db.Conf.Paths.DefaultInDir = "gwIn"
		db.Conf.Paths.DefaultOutDir = "gwOut"
		db.Conf.Paths.DefaultTmpDir = "gwTmp"

		server := &model.LocalAgent{
			Name:        "server",
			Protocol:    testhelpers.TestProtocol,
			Root:        "serRoot",
			LocalInDir:  "serIn",
			LocalOutDir: "serOut",
			LocalTmpDir: "serTmp",
			ProtoConfig: json.RawMessage(`{}`),
			Address:     "localhost:0",
		}
		So(db.Insert(server).Run(), ShouldBeNil)

		acc := &model.LocalAccount{
			LocalAgentID: server.ID,
			Login:        "toto",
			PasswordHash: hash("sesame"),
		}
		So(db.Insert(acc).Run(), ShouldBeNil)

		send := &model.Rule{
			Name:        "SEND",
			IsSend:      true,
			Path:        "/path",
			LocalDir:    "sendLoc",
			RemoteDir:   "sendRem",
			LocalTmpDir: "sendTmp",
		}
		So(db.Insert(send).Run(), ShouldBeNil)

		recv := &model.Rule{
			Name:        "RECEIVE",
			IsSend:      false,
			Path:        "/path",
			LocalDir:    "recvLoc",
			RemoteDir:   "recvRem",
			LocalTmpDir: "recvTmp",
		}
		So(db.Insert(recv).Run(), ShouldBeNil)

		Convey("Given an incoming transfer", func(c C) {
			trans := &model.Transfer{
				RuleID:     recv.ID,
				IsServer:   true,
				AgentID:    server.ID,
				AccountID:  acc.ID,
				LocalPath:  "file.loc",
				RemotePath: "file.rem",
			}
			tmp := trans.LocalPath + ".part"

			transCtx, err := model.GetTransferInfo(db, logger, trans)
			So(err, ShouldBeNil)

			type testCase struct {
				serRoot, ruleLoc, ruleTmp string
				expTmp                    string
			}
			gwRoot := db.Conf.Paths.GatewayHome
			testCases := []testCase{
				{"", "", "", Join(gwRoot, "gwTmp", tmp)},
				{"serRoot", "", "", Join(gwRoot, "serRoot", "serTmp", tmp)},
				{"", "recvLoc", "", Join(gwRoot, "recvLoc", tmp)},
				{"", "", "recvTmp", Join(gwRoot, "recvTmp", tmp)},
				{"serRoot", "recvLoc", "", Join(gwRoot, "serRoot", "recvLoc", tmp)},
				{"serRoot", "", "recvTmp", Join(gwRoot, "serRoot", "recvTmp", tmp)},
				{"", "recvLoc", "recvTmp", Join(gwRoot, "recvTmp", tmp)},
				{"serRoot", "recvLoc", "recvTmp", Join(gwRoot, "serRoot", "recvTmp", tmp)},
			}

			for _, tc := range testCases {
				testCaseName := fmt.Sprintf(
					"Given the following path parameters: %q %q %q",
					tc.serRoot, tc.ruleLoc, tc.ruleTmp,
				)
				Convey(testCaseName, func() {
					transCtx.LocalAgent.Root = tc.serRoot
					if tc.serRoot != "" {
						transCtx.LocalAgent.LocalInDir = "serIn"
						transCtx.LocalAgent.LocalOutDir = "serOut"
						transCtx.LocalAgent.LocalTmpDir = "serTmp"
					} else {
						transCtx.LocalAgent.LocalInDir = ""
						transCtx.LocalAgent.LocalOutDir = ""
						transCtx.LocalAgent.LocalTmpDir = ""
					}

					transCtx.Rule.LocalDir = tc.ruleLoc
					transCtx.Rule.LocalTmpDir = tc.ruleTmp

					Convey("When building the filepath", func() {
						MakeFilepaths(transCtx)

						Convey("Then it should have built the expected tmp path", func() {
							So(transCtx.Transfer.LocalPath, ShouldEqual, tc.expTmp)
						})
					})
				})
			}
		})

		Convey("Given an outgoing transfer", func(c C) {
			trans := &model.Transfer{
				RuleID:     send.ID,
				IsServer:   true,
				AgentID:    server.ID,
				AccountID:  acc.ID,
				LocalPath:  "file.loc",
				RemotePath: "file.rem",
			}

			file := trans.LocalPath

			transCtx, err := model.GetTransferInfo(db, logger, trans)
			So(err, ShouldBeNil)

			type testCase struct {
				serRoot, ruleLoc string
				expFinal         string
			}
			gwRoot := db.Conf.Paths.GatewayHome
			testCases := []testCase{
				{"", "", Join(gwRoot, "gwOut", file)},
				{"serRoot", "", Join(gwRoot, "serRoot", "serOut", file)},
				{"", "sendLoc", Join(gwRoot, "sendLoc", file)},
				{"serRoot", "sendLoc", Join(gwRoot, "serRoot", "sendLoc", file)},
			}

			for _, tc := range testCases {
				testCaseName := fmt.Sprintf(
					"Given the following path parameters: %q %q",
					tc.serRoot, tc.ruleLoc,
				)
				Convey(testCaseName, func() {
					transCtx.LocalAgent.Root = tc.serRoot
					if tc.serRoot != "" {
						transCtx.LocalAgent.LocalInDir = "serIn"
						transCtx.LocalAgent.LocalOutDir = "serOut"
						transCtx.LocalAgent.LocalTmpDir = "serTmp"
					} else {
						transCtx.LocalAgent.LocalInDir = ""
						transCtx.LocalAgent.LocalOutDir = ""
						transCtx.LocalAgent.LocalTmpDir = ""
					}

					transCtx.Rule.LocalDir = tc.ruleLoc

					Convey("When building the filepath", func() {
						MakeFilepaths(transCtx)

						Convey("Then it should have built the expected out path", func() {
							So(transCtx.Transfer.LocalPath, ShouldEqual, tc.expFinal)
						})
					})
				})
			}
		})
	})
}
