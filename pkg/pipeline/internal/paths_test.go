package internal

import (
	"fmt"
	. "path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"
)

func TestPathBuilder(t *testing.T) {

	Convey("Given a Gateway configuration", t, func(c C) {
		ctx := testhelpers.InitServer(c, testhelpers.TestProtocol, nil)
		ctx.DB.Conf.Paths.DefaultInDir = "gwIn"
		ctx.DB.Conf.Paths.DefaultOutDir = "gwOut"
		ctx.DB.Conf.Paths.DefaultTmpDir = "gwTmp"

		Convey("Given an incoming transfer", func(c C) {
			trans := &model.Transfer{
				RuleID:     ctx.ServerPush.ID,
				IsServer:   true,
				AgentID:    ctx.Server.ID,
				AccountID:  ctx.LocAccount.ID,
				LocalPath:  "file.loc",
				RemotePath: "file.rem",
			}
			tmp := trans.LocalPath + ".part"

			transCtx, err := model.GetTransferInfo(ctx.DB, ctx.Logger, trans)
			So(err, ShouldBeNil)

			type testCase struct {
				serRoot, ruleLoc, ruleTmp string
				expTmp                    string
			}
			gwRoot := ctx.Paths.GatewayHome
			testCases := []testCase{
				{"", "", "", Join(gwRoot, "gwTmp", tmp)},
				{"serRoot", "", "", Join(gwRoot, "serRoot", "serTmp", tmp)},
				{"", "ruleLoc", "", Join(gwRoot, "ruleLoc", tmp)},
				{"", "", "ruleTmp", Join(gwRoot, "ruleTmp", tmp)},
				{"serRoot", "ruleLoc", "", Join(gwRoot, "serRoot", "ruleLoc", tmp)},
				{"serRoot", "", "ruleTmp", Join(gwRoot, "serRoot", "ruleTmp", tmp)},
				{"", "ruleLoc", "ruleTmp", Join(gwRoot, "ruleTmp", tmp)},
				{"serRoot", "ruleLoc", "ruleTmp", Join(gwRoot, "serRoot", "ruleTmp", tmp)},
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
				RuleID:     ctx.ServerPull.ID,
				IsServer:   true,
				AgentID:    ctx.Server.ID,
				AccountID:  ctx.LocAccount.ID,
				LocalPath:  "file.loc",
				RemotePath: "file.rem",
			}

			file := trans.LocalPath

			transCtx, err := model.GetTransferInfo(ctx.DB, ctx.Logger, trans)
			So(err, ShouldBeNil)

			type testCase struct {
				serRoot, ruleLoc string
				expFinal         string
			}
			gwRoot := ctx.Paths.GatewayHome
			testCases := []testCase{
				{"", "", Join(gwRoot, "gwOut", file)},
				{"serRoot", "", Join(gwRoot, "serRoot", "serOut", file)},
				{"", "ruleLoc", Join(gwRoot, "ruleLoc", file)},
				{"serRoot", "ruleLoc", Join(gwRoot, "serRoot", "ruleLoc", file)},
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
