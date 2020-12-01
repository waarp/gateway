package r66

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/executor"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"
	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	logConf := conf.LogConfig{
		Level: "DEBUG",
		LogTo: "stdout",
	}
	_ = log.InitBackend(logConf)
}

var testFileContent = []byte("r66 self transfer test file")

type testContext struct {
	logger                   *log.Logger
	db                       *database.DB
	clientPaths, serverPaths pipeline.Paths

	server     *model.LocalAgent
	locAccount *model.LocalAccount
	partner    *model.RemoteAgent
	remAccount *model.RemoteAccount

	cPush, cPull, sPush, sPull *model.Rule

	trans *model.Transfer
}

func (t *testContext) isPush() bool {
	return t.trans.RuleID == t.cPush.ID
}

func initForSelfTransfer(c C) *testContext {
	logger := log.NewLogger("r66_self_transfer")
	db := database.GetTestDatabase()
	home := testhelpers.TempDir(c, "r66_self_transfer")
	port := testhelpers.GetFreePort(c)

	pathConf := conf.PathsConfig{
		GatewayHome:   home,
		InDirectory:   home,
		OutDirectory:  home,
		WorkDirectory: filepath.Join(home, "tmp"),
	}
	clientPaths := pipeline.Paths{PathsConfig: pathConf}

	root := filepath.Join(home, "r66_server_root")
	c.So(os.MkdirAll(root, 0o700), ShouldBeNil)
	c.So(os.MkdirAll(filepath.Join(root, "in"), 0o700), ShouldBeNil)
	c.So(os.MkdirAll(filepath.Join(root, "out"), 0o700), ShouldBeNil)
	c.So(os.MkdirAll(filepath.Join(root, "work"), 0o700), ShouldBeNil)

	serverPaths := pipeline.Paths{
		PathsConfig: pathConf,
		ServerRoot:  root,
		ServerIn:    "in",
		ServerOut:   "out",
		ServerWork:  "work",
	}

	server := &model.LocalAgent{
		Name:        "r66_server",
		Protocol:    "r66",
		Root:        utils.NormalizePath(root),
		ProtoConfig: json.RawMessage(`{"blockSize":50,"serverPassword":"c2VzYW1l"}`),
		Address:     fmt.Sprintf("localhost:%d", port),
	}
	c.So(db.Create(server), ShouldBeNil)

	locAccount := &model.LocalAccount{
		LocalAgentID: server.ID,
		Login:        "toto",
		Password:     []byte("Sesame"),
	}
	c.So(db.Create(locAccount), ShouldBeNil)

	partner := &model.RemoteAgent{
		Name:     "r66_partner",
		Protocol: "r66",
		ProtoConfig: json.RawMessage(`{"blockSize":50, "serverLogin":"r66_server",
			"serverPassword":"c2VzYW1l"}`),
		Address: fmt.Sprintf("localhost:%d", port),
	}
	c.So(db.Create(partner), ShouldBeNil)

	remAccount := &model.RemoteAccount{
		RemoteAgentID: partner.ID,
		Login:         "toto",
		Password:      []byte("Sesame"),
	}
	c.So(db.Create(remAccount), ShouldBeNil)

	service := NewService(db, server, logger)
	c.So(service.Start(), ShouldBeNil)
	service.server.AuthentHandler = &testAuthHandler{service.server.AuthentHandler}
	c.Reset(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		c.So(service.Stop(ctx), ShouldBeNil)
	})

	service.paths = pipeline.Paths{
		PathsConfig: pathConf,
		ServerRoot:  server.Root,
		ServerIn:    server.InDir,
		ServerOut:   server.OutDir,
		ServerWork:  server.WorkDir,
	}

	return &testContext{
		logger:      logger,
		db:          db,
		clientPaths: clientPaths,
		serverPaths: serverPaths,
		server:      server,
		locAccount:  locAccount,
		partner:     partner,
		remAccount:  remAccount,
		cPush:       makeClientPushRule(c, db),
		cPull:       makeClientPullRule(c, db),
		sPush:       makeServerPushRule(c, db),
		sPull:       makeServerPullRule(c, db),
	}
}

func makeTransfer(c C, ctx *testContext, isPush bool) {
	if isPush {
		testFile := filepath.Join(ctx.clientPaths.GatewayHome, "r66_self_transfer_push.src")
		c.So(ioutil.WriteFile(testFile, testFileContent, 0o600), ShouldBeNil)

		trans := &model.Transfer{
			RuleID:       ctx.cPush.ID,
			IsServer:     false,
			AgentID:      ctx.server.ID,
			AccountID:    ctx.locAccount.ID,
			TrueFilepath: testFile,
			SourceFile:   "r66_self_transfer_push.src",
			DestFile:     "r66_self_transfer_push.dst",
			Start:        time.Now(),
		}
		So(ctx.db.Create(trans), ShouldBeNil)

		ctx.trans = trans
		return
	}

	testFile := filepath.Join(ctx.serverPaths.ServerRoot,
		ctx.serverPaths.ServerOut, "r66_self_transfer_pull.src")
	c.So(ioutil.WriteFile(testFile, testFileContent, 0o600), ShouldBeNil)

	trans := &model.Transfer{
		RuleID:       ctx.cPull.ID,
		IsServer:     false,
		AgentID:      ctx.server.ID,
		AccountID:    ctx.locAccount.ID,
		TrueFilepath: testFile,
		SourceFile:   "r66_self_transfer_pull.src",
		DestFile:     "r66_self_transfer_pull.dst",
		Start:        time.Now(),
	}
	c.So(ctx.db.Create(trans), ShouldBeNil)

	ctx.trans = trans
}

func processTransfer(c C, ctx *testContext) {
	stream, err := pipeline.NewTransferStream(context.Background(),
		ctx.logger, ctx.db, ctx.clientPaths, ctx.trans)
	c.So(err, ShouldBeNil)

	exe := executor.Executor{TransferStream: stream}
	clientCheckChannel = make(chan string, 10)
	serverCheckChannel = make(chan string, 10)
	c.Reset(func() {
		if clientCheckChannel != nil {
			close(clientCheckChannel)
		}
		if serverCheckChannel != nil {
			close(serverCheckChannel)
		}
		clientCheckChannel = nil
		serverCheckChannel = nil
	})
	exe.Run()
	clientCheckChannel <- "CLIENT END TRANSFER"
}

func checkTransfersOK(c C, ctx *testContext) {
	c.Convey("Then the transfers should be over", func() {
		var transfers []model.Transfer
		So(ctx.db.Select(&transfers, nil), ShouldBeNil)
		So(transfers, ShouldBeEmpty)

		var results []model.TransferHistory
		So(ctx.db.Select(&results, nil), ShouldBeNil)
		So(len(results), ShouldEqual, 2)

		Convey("Then there should be a client-side history entry", func() {
			cTrans := model.TransferHistory{
				ID:             ctx.trans.ID,
				Owner:          database.Owner,
				Protocol:       "r66",
				IsServer:       false,
				Account:        ctx.remAccount.Login,
				Agent:          ctx.partner.Name,
				Start:          results[0].Start,
				Stop:           results[0].Stop,
				SourceFilename: ctx.trans.SourceFile,
				DestFilename:   ctx.trans.DestFile,
				Status:         types.StatusDone,
				Step:           types.StepNone,
				Error:          types.TransferError{},
				Progress:       uint64(len(testFileContent)),
				TaskNumber:     0,
			}
			if ctx.isPush() {
				cTrans.IsSend = true
				cTrans.Rule = ctx.cPush.Name
			} else {
				cTrans.IsSend = false
				cTrans.Rule = ctx.cPull.Name
			}
			So(results[0], ShouldResemble, cTrans)
		})

		Convey("Then there should be a server-side history entry", func() {
			sTrans := model.TransferHistory{
				ID:               ctx.trans.ID + 1,
				RemoteTransferID: fmt.Sprint(ctx.trans.ID),
				Owner:            database.Owner,
				Protocol:         "r66",
				IsServer:         true,
				Account:          ctx.locAccount.Login,
				Agent:            ctx.server.Name,
				Start:            results[1].Start,
				Stop:             results[1].Stop,
				Status:           types.StatusDone,
				Step:             types.StepNone,
				Error:            types.TransferError{},
				Progress:         uint64(len(testFileContent)),
				TaskNumber:       0,
			}
			if ctx.isPush() {
				sTrans.IsSend = false
				sTrans.Rule = ctx.sPush.Name
				sTrans.SourceFilename = ctx.trans.DestFile
				sTrans.DestFilename = ctx.trans.DestFile
			} else {
				sTrans.IsSend = true
				sTrans.Rule = ctx.sPull.Name
				sTrans.SourceFilename = ctx.trans.SourceFile
				sTrans.DestFilename = ctx.trans.SourceFile
			}
			So(results[1], ShouldResemble, sTrans)
		})
	})

	checkDestFile(c, ctx)
}

func checkTransfersErr(c C, ctx *testContext, cTrans *model.Transfer, sTrans ...*model.Transfer) {
	c.Convey("Then the transfers should be over", func() {
		var results []model.Transfer
		So(ctx.db.Select(&results, nil), ShouldBeNil)
		So(len(results), ShouldEqual, 2)

		Convey("Then there should be a client-side transfer entry in error", func() {
			cTrans.Owner = database.Owner
			cTrans.ID = ctx.trans.ID
			cTrans.Status = types.StatusError
			cTrans.IsServer = false
			cTrans.RuleID = ctx.trans.RuleID
			cTrans.AccountID = ctx.remAccount.ID
			cTrans.AgentID = ctx.partner.ID
			cTrans.Start = results[0].Start
			cTrans.SourceFile = ctx.trans.SourceFile
			cTrans.DestFile = ctx.trans.DestFile
			cTrans.TrueFilepath = ctx.trans.TrueFilepath
			So(results[0], ShouldResemble, *cTrans)
		})

		Convey("Then there should be a server-side transfer entry in error", func() {
			var transfers []interface{}
			for _, sTrans := range sTrans {
				sTrans.Owner = database.Owner
				sTrans.ID = ctx.trans.ID + 1
				sTrans.Status = types.StatusError
				sTrans.RemoteTransferID = fmt.Sprint(ctx.trans.ID)
				sTrans.IsServer = true
				sTrans.AccountID = ctx.locAccount.ID
				sTrans.AgentID = ctx.server.ID
				sTrans.Start = results[1].Start
				if ctx.isPush() {
					sTrans.RuleID = ctx.sPush.ID
					sTrans.SourceFile = ctx.trans.DestFile
					sTrans.DestFile = ctx.trans.DestFile
					if sTrans.Step > types.StepData {
						sTrans.TrueFilepath = utils.NormalizePath(filepath.Join(
							ctx.serverPaths.ServerRoot, ctx.serverPaths.ServerIn,
							ctx.trans.DestFile))
					} else {
						sTrans.TrueFilepath = utils.NormalizePath(filepath.Join(
							ctx.serverPaths.ServerRoot, ctx.serverPaths.ServerWork,
							ctx.trans.DestFile+".tmp"))
					}
				} else {
					sTrans.RuleID = ctx.sPull.ID
					sTrans.SourceFile = ctx.trans.SourceFile
					sTrans.DestFile = ctx.trans.SourceFile
					sTrans.TrueFilepath = utils.NormalizePath(filepath.Join(
						ctx.serverPaths.ServerRoot, ctx.serverPaths.ServerOut,
						ctx.trans.SourceFile))
				}
				transfers = append(transfers, *sTrans)
			}
			if len(sTrans) > 1 {
				So(results[1], testhelpers.ShouldBeOneOf, transfers...)
			} else {
				So(results[1], ShouldResemble, *sTrans[0])
			}
		})
	})
}

func checkDestFile(c C, ctx *testContext) {
	c.Convey("Then the file should have been sent entirely", func() {
		path := filepath.Join(ctx.clientPaths.GatewayHome, ctx.trans.DestFile)
		if ctx.isPush() {
			path = filepath.Join(ctx.serverPaths.ServerRoot, ctx.serverPaths.ServerIn,
				ctx.trans.DestFile)
		}
		content, err := ioutil.ReadFile(path)
		So(err, ShouldBeNil)
		So(string(content), ShouldEqual, string(testFileContent))
	})
}

func addClientFailure(c C, ctx *testContext, chain model.Chain) (string, func()) {
	return addTaskFailure(c, ctx, true, chain)
}

func addServerFailure(c C, ctx *testContext, chain model.Chain) (string, func()) {
	return addTaskFailure(c, ctx, false, chain)
}

func addTaskFailure(c C, ctx *testContext, isClient bool, chain model.Chain) (string, func()) {
	taskFailure := &model.Task{
		Rank:  1,
		Chain: chain,
	}
	var errMsg string

	if isClient {
		taskFailure.Type = clientErr
		if ctx.isPush() {
			taskFailure.RuleID = ctx.cPush.ID
			taskFailure.Args = json.RawMessage(`{"msg":"PUSH | ` + chain + `-TASKS[1]"}`)
			errMsg = "Task " + clientErr + " @ " + ctx.cPush.Name + " " + string(chain) + "[1]: task failed"
		} else {
			taskFailure.RuleID = ctx.cPull.ID
			taskFailure.Args = json.RawMessage(`{"msg":"PULL | ` + chain + `-TASKS[1]"}`)
			errMsg = "Task " + clientErr + " @ " + ctx.cPull.Name + " " + string(chain) + "[1]: task failed"
		}
	} else {
		taskFailure.Type = serverErr
		if ctx.isPush() {
			taskFailure.RuleID = ctx.sPush.ID
			taskFailure.Args = json.RawMessage(`{"msg":"PUSH | ` + chain + `-TASKS[1]"}`)
			errMsg = "Task " + serverErr + " @ " + ctx.sPush.Name + " " + string(chain) + "[1]: task failed"
		} else {
			taskFailure.RuleID = ctx.sPull.ID
			taskFailure.Args = json.RawMessage(`{"msg":"PULL | ` + chain + `-TASKS[1]"}`)
			errMsg = "Task " + serverErr + " @ " + ctx.sPull.Name + " " + string(chain) + "[1]: task failed"
		}
	}
	c.So(ctx.db.Create(taskFailure), ShouldBeNil)

	return errMsg, func() {
		c.So(ctx.db.Delete(taskFailure), ShouldBeNil)
	}
}

func retryTransfer(c C, ctx *testContext, removeFail func()) {
	removeFail()
	retry := &model.Transfer{ID: ctx.trans.ID}
	c.So(ctx.db.Get(retry), ShouldBeNil)
	retry.Status = types.StatusPlanned
	c.So(ctx.db.Update(retry), ShouldBeNil)
	ctx.trans = retry
}

func makeClientPushRule(c C, db *database.DB) *model.Rule {
	clientPush := &model.Rule{
		Name:   "push",
		Path:   "/push",
		IsSend: true,
	}
	c.So(db.Create(clientPush), ShouldBeNil)

	cPreTask := &model.Task{
		RuleID: clientPush.ID,
		Chain:  model.ChainPre,
		Rank:   0,
		Type:   clientOK,
		Args:   json.RawMessage(`{"msg":"PUSH | PRE-TASKS[0]"}`),
	}
	c.So(db.Create(cPreTask), ShouldBeNil)

	cPostTask := &model.Task{
		RuleID: clientPush.ID,
		Chain:  model.ChainPost,
		Rank:   0,
		Type:   clientOK,
		Args:   json.RawMessage(`{"msg":"PUSH | POST-TASKS[0]"}`),
	}
	c.So(db.Create(cPostTask), ShouldBeNil)

	cErrTask := &model.Task{
		RuleID: clientPush.ID,
		Chain:  model.ChainError,
		Rank:   0,
		Type:   clientOK,
		Args:   json.RawMessage(`{"msg":"PUSH | ERROR-TASKS[0]"}`),
	}
	c.So(db.Create(cErrTask), ShouldBeNil)

	return clientPush
}

func makeServerPushRule(c C, db *database.DB) *model.Rule {
	serverPush := &model.Rule{
		Name:   "push",
		Path:   "/push",
		IsSend: false,
	}
	c.So(db.Create(serverPush), ShouldBeNil)

	sPreTask := &model.Task{
		RuleID: serverPush.ID,
		Chain:  model.ChainPre,
		Rank:   0,
		Type:   serverOK,
		Args:   json.RawMessage(`{"msg":"PUSH | PRE-TASKS[0]"}`),
	}
	c.So(db.Create(sPreTask), ShouldBeNil)

	sPostTask := &model.Task{
		RuleID: serverPush.ID,
		Chain:  model.ChainPost,
		Rank:   0,
		Type:   serverOK,
		Args:   json.RawMessage(`{"msg":"PUSH | POST-TASKS[0]"}`),
	}
	c.So(db.Create(sPostTask), ShouldBeNil)

	cErrTask := &model.Task{
		RuleID: serverPush.ID,
		Chain:  model.ChainError,
		Rank:   0,
		Type:   serverOK,
		Args:   json.RawMessage(`{"msg":"PUSH | ERROR-TASKS[0]"}`),
	}
	c.So(db.Create(cErrTask), ShouldBeNil)

	return serverPush
}

func makeClientPullRule(c C, db *database.DB) *model.Rule {
	clientPull := &model.Rule{
		Name:   "pull",
		Path:   "/pull",
		IsSend: false,
	}
	c.So(db.Create(clientPull), ShouldBeNil)

	cPreTask := &model.Task{
		RuleID: clientPull.ID,
		Chain:  model.ChainPre,
		Rank:   0,
		Type:   clientOK,
		Args:   json.RawMessage(`{"msg":"PULL | PRE-TASKS[0]"}`),
	}
	c.So(db.Create(cPreTask), ShouldBeNil)

	cPostTask := &model.Task{
		RuleID: clientPull.ID,
		Chain:  model.ChainPost,
		Rank:   0,
		Type:   clientOK,
		Args:   json.RawMessage(`{"msg":"PULL | POST-TASKS[0]"}`),
	}
	c.So(db.Create(cPostTask), ShouldBeNil)

	cErrTask := &model.Task{
		RuleID: clientPull.ID,
		Chain:  model.ChainError,
		Rank:   0,
		Type:   clientOK,
		Args:   json.RawMessage(`{"msg":"PULL | ERROR-TASKS[0]"}`),
	}
	c.So(db.Create(cErrTask), ShouldBeNil)

	return clientPull
}

func makeServerPullRule(c C, db *database.DB) *model.Rule {
	serverPull := &model.Rule{
		Name:   "pull",
		Path:   "/pull",
		IsSend: true,
	}
	c.So(db.Create(serverPull), ShouldBeNil)

	sPreTask := &model.Task{
		RuleID: serverPull.ID,
		Chain:  model.ChainPre,
		Rank:   0,
		Type:   serverOK,
		Args:   json.RawMessage(`{"msg":"PULL | PRE-TASKS[0]"}`),
	}
	c.So(db.Create(sPreTask), ShouldBeNil)

	sPostTask := &model.Task{
		RuleID: serverPull.ID,
		Chain:  model.ChainPost,
		Rank:   0,
		Type:   serverOK,
		Args:   json.RawMessage(`{"msg":"PULL | POST-TASKS[0]"}`),
	}
	c.So(db.Create(sPostTask), ShouldBeNil)

	cErrTask := &model.Task{
		RuleID: serverPull.ID,
		Chain:  model.ChainError,
		Rank:   0,
		Type:   serverOK,
		Args:   json.RawMessage(`{"msg":"PULL | ERROR-TASKS[0]"}`),
	}
	c.So(db.Create(cErrTask), ShouldBeNil)

	return serverPull
}
