package internal

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

const testProtocol = "test_proto"

//nolint:gochecknoinits // init is used to ease the tests
func init() {
	_ = log.InitBackend("DEBUG", "stdout", "")

	config.ProtoConfigs[testProtocol] = func() config.ProtoConfig {
		return new(testhelpers.TestProtoConfig)
	}
}

func hash(pwd string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.MinCost)
	So(err, ShouldBeNil)

	return string(h)
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
			Protocol:    testProtocol,
			Root:        "serRoot",
			InDir:       "serIn",
			OutDir:      "serOut",
			TmpDir:      "serTmp",
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
			gwRoot := db.Conf.Paths.GatewayHome
			testCases := []testCase{
				{"", "", "", filepath.Join(gwRoot, "gwTmp", file)},
				{"serRoot", "", "", filepath.Join(gwRoot, "serRoot", "serTmp", file)},
				{"", "recvLoc", "", filepath.Join(gwRoot, "recvLoc", file)},
				{"", "", "recvTmp", filepath.Join(gwRoot, "recvTmp", file)},
				{"serRoot", "recvLoc", "", filepath.Join(gwRoot, "serRoot", "recvLoc", file)},
				{"serRoot", "", "recvTmp", filepath.Join(gwRoot, "serRoot", "recvTmp", file)},
				{"", "recvLoc", "recvTmp", filepath.Join(gwRoot, "recvTmp", file)},
				{"serRoot", "recvLoc", "recvTmp", filepath.Join(gwRoot, "serRoot", "recvTmp", file)},
			}

			for _, tc := range testCases {
				testCaseName := fmt.Sprintf(
					"Given the following path parameters: %q %q %q",
					tc.serRoot, tc.ruleLoc, tc.ruleTmp,
				)
				Convey(testCaseName, func() {
					transCtx.LocalAgent.Root = tc.serRoot
					if tc.serRoot != "" {
						transCtx.LocalAgent.InDir = "serIn"
						transCtx.LocalAgent.OutDir = "serOut"
						transCtx.LocalAgent.TmpDir = "serTmp"
					} else {
						transCtx.LocalAgent.InDir = ""
						transCtx.LocalAgent.OutDir = ""
						transCtx.LocalAgent.TmpDir = ""
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
			gwRoot := db.Conf.Paths.GatewayHome
			testCases := []testCase{
				{"", "", filepath.Join(gwRoot, "gwOut", file)},
				{"serRoot", "", filepath.Join(gwRoot, "serRoot", "serOut", file)},
				{"", "sendLoc", filepath.Join(gwRoot, "sendLoc", file)},
				{"serRoot", "sendLoc", filepath.Join(gwRoot, "serRoot", "sendLoc", file)},
			}

			for _, tc := range testCases {
				testCaseName := fmt.Sprintf(
					"Given the following path parameters: %q %q",
					tc.serRoot, tc.ruleLoc,
				)
				Convey(testCaseName, func() {
					transCtx.LocalAgent.Root = tc.serRoot
					if tc.serRoot != "" {
						transCtx.LocalAgent.InDir = "serIn"
						transCtx.LocalAgent.OutDir = "serOut"
						transCtx.LocalAgent.TmpDir = "serTmp"
					} else {
						transCtx.LocalAgent.InDir = ""
						transCtx.LocalAgent.OutDir = ""
						transCtx.LocalAgent.TmpDir = ""
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
