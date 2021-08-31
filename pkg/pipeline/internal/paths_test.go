package internal

import (
	"encoding/json"
	"fmt"
	. "path/filepath"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"

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
		conf.GlobalConfig.Paths.GatewayHome = testhelpers.TempDir(c, "path_builder")
		conf.GlobalConfig.Paths.DefaultInDir = "gwIn"
		conf.GlobalConfig.Paths.DefaultOutDir = "gwOut"
		conf.GlobalConfig.Paths.DefaultTmpDir = "gwTmp"

		server := &model.LocalAgent{
			Name:        "server",
			Protocol:    config.TestProtocol,
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
			file := trans.LocalPath

			transCtx, err := model.GetTransferContext(db, logger, trans)
			So(err, ShouldBeNil)

			type testCase struct {
				serRoot, ruleLoc, ruleTmp string
				expTmp                    string
			}
			gwRoot := conf.GlobalConfig.Paths.GatewayHome
			testCases := []testCase{
				{"", "", "", Join(gwRoot, "gwTmp", file)},
				{"serRoot", "", "", Join(gwRoot, "serRoot", "serTmp", file)},
				{"", "recvLoc", "", Join(gwRoot, "recvLoc", file)},
				{"", "", "recvTmp", Join(gwRoot, "recvTmp", file)},
				{"serRoot", "recvLoc", "", Join(gwRoot, "serRoot", "recvLoc", file)},
				{"serRoot", "", "recvTmp", Join(gwRoot, "serRoot", "recvTmp", file)},
				{"", "recvLoc", "recvTmp", Join(gwRoot, "recvTmp", file)},
				{"serRoot", "recvLoc", "recvTmp", Join(gwRoot, "serRoot", "recvTmp", file)},
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

			transCtx, err := model.GetTransferContext(db, logger, trans)
			So(err, ShouldBeNil)

			type testCase struct {
				serRoot, ruleLoc string
				expFinal         string
			}
			gwRoot := conf.GlobalConfig.Paths.GatewayHome
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
