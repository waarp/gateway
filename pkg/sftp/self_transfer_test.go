package sftp

import (
	"encoding/json"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers/selftransfer"

	. "github.com/smartystreets/goconvey/convey"
)

var (
	servConf = json.RawMessage("{}")
	partConf = json.RawMessage("{}")
)

func TestSelfPushOK(t *testing.T) {
	Convey("Given an SFTP service", t, func(c C) {
		ctx := selftransfer.InitDBForSelfTransfer(c, "sftp", servConf, partConf)
		addCerts(c, ctx)
		startService(c, ctx)

		Convey("Given a new SFTP push transfer", func(c C) {
			selftransfer.AddTransfer(c, ctx, true)

			Convey("Once the transfer has been processed", func(c C) {
				selftransfer.RunTransfer(c, ctx)

				Convey("Then it should have executed all the tasks in order", func(c C) {
					testhelpers.ServerMsgShouldBe(c, "SERVER | PUSH | PRE-TASKS[0] | OK")
					testhelpers.ClientMsgShouldBe(c, "CLIENT | PUSH | PRE-TASKS[0] | OK")
					testhelpers.ServerMsgShouldBe(c, "SERVER | PUSH | POST-TASKS[0] | OK")
					testhelpers.ClientMsgShouldBe(c, "CLIENT | PUSH | POST-TASKS[0] | OK")
					testhelpers.ServerMsgShouldBe(c, "SERVER TRANSFER END")
					testhelpers.ClientMsgShouldBe(c, "CLIENT TRANSFER END")

					selftransfer.CheckTransfersOK(c, ctx)
				})
			})
		})
	})
}

func TestSelfPullOK(t *testing.T) {
	Convey("Given an SFTP service", t, func(c C) {
		ctx := selftransfer.InitDBForSelfTransfer(c, "sftp", servConf, partConf)
		addCerts(c, ctx)
		startService(c, ctx)

		Convey("Given a new SFTP pull transfer", func(c C) {
			selftransfer.AddTransfer(c, ctx, false)

			Convey("Once the transfer has been processed", func(c C) {
				selftransfer.RunTransfer(c, ctx)

				Convey("Then it should have executed all the tasks in order", func(c C) {
					testhelpers.ServerMsgShouldBe(c, "SERVER | PULL | PRE-TASKS[0] | OK")
					testhelpers.ClientMsgShouldBe(c, "CLIENT | PULL | PRE-TASKS[0] | OK")
					testhelpers.ServerMsgShouldBe(c, "SERVER | PULL | POST-TASKS[0] | OK")
					testhelpers.ClientMsgShouldBe(c, "CLIENT | PULL | POST-TASKS[0] | OK")
					testhelpers.ServerMsgShouldBe(c, "SERVER TRANSFER END")
					testhelpers.ClientMsgShouldBe(c, "CLIENT TRANSFER END")

					selftransfer.CheckTransfersOK(c, ctx)
				})
			})
		})
	})
}

func TestSelfErrorClient(t *testing.T) {
	Convey("Given an SFTP service", t, func(c C) {
		ctx := selftransfer.InitDBForSelfTransfer(c, "sftp", servConf, partConf)
		addCerts(c, ctx)
		startService(c, ctx)

		Convey("Given a new SFTP push transfer", func(c C) {
			selftransfer.AddTransfer(c, ctx, true)

			Convey("Given that an error occurs", func(c C) {
				task := model.Task{
					RuleID: ctx.ClientPush.ID,
					Chain:  model.ChainPre,
					Rank:   1,
					Type:   testhelpers.ClientErr,
					Args:   json.RawMessage(`{"msg":"PUSH | PRE-TASKS[1]"}`),
				}
				So(ctx.DB.Insert(&task).Run(), ShouldBeNil)

				Convey("Once the transfer has been processed", func(c C) {
					selftransfer.RunTransfer(c, ctx)

					Convey("Then it should have executed all the tasks in order", func(c C) {
						testhelpers.ServerMsgShouldBe(c, "SERVER | PUSH | PRE-TASKS[0] | OK")
						testhelpers.ClientMsgShouldBe(c, "CLIENT | PUSH | PRE-TASKS[0] | OK")
						testhelpers.ClientMsgShouldBe(c, "CLIENT | PUSH | PRE-TASKS[1] | ERROR")
						testhelpers.ClientMsgShouldBe(c, "CLIENT | PUSH | ERROR-TASKS[0] | OK")
						testhelpers.ServerMsgShouldBe(c, "SERVER | PUSH | ERROR-TASKS[0] | OK")
						testhelpers.ServerMsgShouldBe(c, "SERVER TRANSFER END")
						testhelpers.ClientMsgShouldBe(c, "CLIENT TRANSFER END")

						cTrans := &model.Transfer{
							Step:       types.StepPreTasks,
							Progress:   0,
							TaskNumber: 1,
							Error: types.NewTransferError(types.TeExternalOperation,
								"Pre-tasks failed: Task CLIENTERR @ PUSH PRE[1]: task failed"),
						}
						sTrans := &model.Transfer{
							Step:       types.StepPreTasks,
							Progress:   0,
							TaskNumber: 0,
							Error: types.NewTransferError(types.TeUnknownRemote,
								"An error occurred on the remote partner: "+
									"session closed by remote host"),
						}

						selftransfer.CheckTransfersError(c, ctx, cTrans, sTrans)
					})
				})
			})
		})
	})
}

func TestSelfErrorServer(t *testing.T) {
	Convey("Given an SFTP service", t, func(c C) {
		ctx := selftransfer.InitDBForSelfTransfer(c, "sftp", servConf, partConf)
		addCerts(c, ctx)
		startService(c, ctx)

		Convey("Given a new SFTP push transfer", func(c C) {
			selftransfer.AddTransfer(c, ctx, true)

			Convey("Given that an error occurs", func(c C) {
				task := model.Task{
					RuleID: ctx.ServerPush.ID,
					Chain:  model.ChainPre,
					Rank:   1,
					Type:   testhelpers.ServerErr,
					Args:   json.RawMessage(`{"msg":"PUSH | PRE-TASKS[1]"}`),
				}
				So(ctx.DB.Insert(&task).Run(), ShouldBeNil)

				Convey("Once the transfer has been processed", func(c C) {
					selftransfer.RunTransfer(c, ctx)

					Convey("Then it should have executed all the tasks in order", func(c C) {
						testhelpers.ServerMsgShouldBe(c, "SERVER | PUSH | PRE-TASKS[0] | OK")
						testhelpers.ServerMsgShouldBe(c, "SERVER | PUSH | PRE-TASKS[1] | ERROR")
						testhelpers.ServerMsgShouldBe(c, "SERVER | PUSH | ERROR-TASKS[0] | OK")
						testhelpers.ClientMsgShouldBe(c, "CLIENT | PUSH | ERROR-TASKS[0] | OK")
						testhelpers.ServerMsgShouldBe(c, "SERVER TRANSFER END")
						testhelpers.ClientMsgShouldBe(c, "CLIENT TRANSFER END")

						cTrans := &model.Transfer{
							Step:       types.StepSetup,
							Progress:   0,
							TaskNumber: 0,
							Error: types.NewTransferError(types.TeUnknownRemote,
								"An error occurred on the remote partner: "+
									"failed to create remote file"),
						}
						sTrans := &model.Transfer{
							Step:       types.StepPreTasks,
							Progress:   0,
							TaskNumber: 1,
							Error: types.NewTransferError(types.TeExternalOperation,
								"Pre-tasks failed: Task SERVERERR @ PUSH PRE[1]: task failed"),
						}

						selftransfer.CheckTransfersError(c, ctx, cTrans, sTrans)
					})
				})
			})
		})
	})
}

func TestSelfPushRetry(t *testing.T) {
	Convey("Given an SFTP service", t, func(c C) {
		ctx := selftransfer.InitDBForSelfTransfer(c, "sftp", servConf, partConf)
		addCerts(c, ctx)
		startService(c, ctx)

		Convey("Given a failed SFTP push transfer", func(c C) {
			selftransfer.AddTransfer(c, ctx, true)
			task := model.Task{
				RuleID: ctx.ClientPush.ID,
				Chain:  model.ChainPost,
				Rank:   1,
				Type:   testhelpers.ClientErr,
				Args:   json.RawMessage(`{"msg":"PUSH | POST-TASKS[1]"}`),
			}
			So(ctx.DB.Insert(&task).Run(), ShouldBeNil)

			selftransfer.RunTransfer(c, ctx)
			testhelpers.ServerMsgShouldBe(c, "SERVER | PUSH | PRE-TASKS[0] | OK")
			testhelpers.ClientMsgShouldBe(c, "CLIENT | PUSH | PRE-TASKS[0] | OK")
			testhelpers.ClientMsgShouldBe(c, "CLIENT | PUSH | POST-TASKS[0] | OK")
			testhelpers.ClientMsgShouldBe(c, "CLIENT | PUSH | POST-TASKS[1] | ERROR")
			testhelpers.ServerMsgShouldBe(c, "SERVER | PUSH | ERROR-TASKS[0] | OK")
			testhelpers.ClientMsgShouldBe(c, "CLIENT | PUSH | ERROR-TASKS[0] | OK")
			testhelpers.ServerMsgShouldBe(c, "SERVER TRANSFER END")
			testhelpers.ClientMsgShouldBe(c, "CLIENT TRANSFER END")

			Convey("When retrying the transfer", func(c C) {
				So(ctx.DB.DeleteAll(&task).Where("type=?", testhelpers.ClientErr).
					Run(), ShouldBeNil)
				ctx.Trans.Status = types.StatusPlanned

				Convey("Once the transfer has been processed", func(c C) {
					selftransfer.RunTransfer(c, ctx)

					Convey("Then it should have executed all the remaining tasks in order", func(c C) {
						testhelpers.ServerMsgShouldBe(c, "SERVER | PUSH | PRE-TASKS[0] | OK")
						testhelpers.ServerMsgShouldBe(c, "SERVER | PUSH | POST-TASKS[0] | OK")
						testhelpers.ServerMsgShouldBe(c, "SERVER TRANSFER END")
						testhelpers.ClientMsgShouldBe(c, "CLIENT TRANSFER END")

						selftransfer.CheckTransfersOK(c, ctx)
					})
				})
			})
		})
	})
}
