package selftransfer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
)

var (
	testFileContent = []byte("self transfer test file")
)

// Context is a struct regrouping all the elements necessary for a self-transfer
// test.
type Context struct {
	Logger *log.Logger
	DB     *database.DB
	Paths  *conf.PathsConfig

	Server     *model.LocalAgent
	LocAccount *model.LocalAccount
	Partner    *model.RemoteAgent
	RemAccount *model.RemoteAccount

	ServerCerts, PartnerCerts, LocAccCerts, RemAccCerts model.Certificates

	ClientPush, ClientPull, ServerPush, ServerPull *model.Rule

	Trans *model.Transfer
}

func (c *Context) isPush() bool {
	return c.Trans.RuleID == c.ClientPush.ID
}

// InitDBForSelfTransfer creates a database and fills it with all the elements
// necessary for a self-transfer test of the given protocol. It then returns all
// these element inside a Context.
func InitDBForSelfTransfer(c C, proto string, partConf, servConf json.RawMessage) *Context {
	logger := log.NewLogger(proto + "_self_transfer")
	db := database.TestDatabase(c, "ERROR")
	home := testhelpers.TempDir(c, proto+"_self_transfer")
	port := testhelpers.GetFreePort(c)

	paths := &conf.PathsConfig{
		GatewayHome:   home,
		DefaultInDir:  "in",
		DefaultOutDir: "out",
		DefaultTmpDir: "tmp",
	}
	c.So(os.MkdirAll(filepath.Join(home, paths.DefaultInDir), 0o700), ShouldBeNil)
	c.So(os.MkdirAll(filepath.Join(home, paths.DefaultOutDir), 0o700), ShouldBeNil)
	c.So(os.MkdirAll(filepath.Join(home, paths.DefaultTmpDir), 0o700), ShouldBeNil)

	root := filepath.Join(home, proto+"_server_root")
	c.So(os.MkdirAll(root, 0o700), ShouldBeNil)

	server := &model.LocalAgent{
		Name:        "server",
		Protocol:    proto,
		Root:        utils.ToStandardPath(root),
		ProtoConfig: servConf,
		Address:     fmt.Sprintf("localhost:%d", port),
	}
	c.So(db.Insert(server).Run(), ShouldBeNil)
	c.So(os.MkdirAll(filepath.Join(root, server.LocalInDir), 0o700), ShouldBeNil)
	c.So(os.MkdirAll(filepath.Join(root, server.LocalOutDir), 0o700), ShouldBeNil)
	c.So(os.MkdirAll(filepath.Join(root, server.LocalTmpDir), 0o700), ShouldBeNil)

	locAccount := &model.LocalAccount{
		LocalAgentID: server.ID,
		Login:        "toto",
		Password:     []byte("sesame"),
	}
	c.So(db.Insert(locAccount).Run(), ShouldBeNil)

	partner := &model.RemoteAgent{
		Name:        "partner",
		Protocol:    proto,
		ProtoConfig: partConf,
		Address:     fmt.Sprintf("localhost:%d", port),
	}
	c.So(db.Insert(partner).Run(), ShouldBeNil)

	remAccount := &model.RemoteAccount{
		RemoteAgentID: partner.ID,
		Login:         "toto",
		Password:      []byte("sesame"),
	}
	c.So(db.Insert(remAccount).Run(), ShouldBeNil)

	return &Context{
		Logger:     logger,
		DB:         db,
		Paths:      paths,
		Server:     server,
		LocAccount: locAccount,
		Partner:    partner,
		RemAccount: remAccount,
		ClientPush: makeRule(c, db, true, false),
		ClientPull: makeRule(c, db, false, false),
		ServerPush: makeRule(c, db, true, true),
		ServerPull: makeRule(c, db, false, true),
	}
}

func makeRule(c C, db *database.DB, isPush, isServer bool) *model.Rule {
	var taskType string
	rule := &model.Rule{}

	if isServer {
		taskType = testhelpers.ServerOK
		if isPush {
			rule.Name = "PUSH"
			rule.IsSend = false
			rule.Path = "/push"
		} else {
			rule.Name = "PULL"
			rule.IsSend = true
			rule.Path = "/pull"
		}
	} else {
		taskType = testhelpers.ClientOK
		if isPush {
			rule.Name = "PUSH"
			rule.IsSend = true
			rule.Path = "/push"
			rule.RemoteDir = "/push"
		} else {
			rule.Name = "PULL"
			rule.IsSend = false
			rule.Path = "/pull"
			rule.RemoteDir = "/pull"
		}
	}

	c.So(db.Insert(rule).Run(), ShouldBeNil)

	cPreTask := &model.Task{
		RuleID: rule.ID,
		Chain:  model.ChainPre,
		Rank:   0,
		Type:   taskType,
		Args:   json.RawMessage(`{"msg":"` + rule.Name + ` | PRE-TASKS[0]"}`),
	}
	c.So(db.Insert(cPreTask).Run(), ShouldBeNil)

	cPostTask := &model.Task{
		RuleID: rule.ID,
		Chain:  model.ChainPost,
		Rank:   0,
		Type:   taskType,
		Args:   json.RawMessage(`{"msg":"` + rule.Name + ` | POST-TASKS[0]"}`),
	}
	c.So(db.Insert(cPostTask).Run(), ShouldBeNil)

	cErrTask := &model.Task{
		RuleID: rule.ID,
		Chain:  model.ChainError,
		Rank:   0,
		Type:   taskType,
		Args:   json.RawMessage(`{"msg":"` + rule.Name + ` | ERROR-TASKS[0]"}`),
	}
	c.So(db.Insert(cErrTask).Run(), ShouldBeNil)

	return rule
}

// AddTransfer creates a new transfer with the given direction and adds it to
// the database. The transfer is then added to the Context.
func AddTransfer(c C, ctx *Context, isPush bool) {
	if isPush {
		testFile := filepath.Join(ctx.Paths.GatewayHome, ctx.Paths.DefaultOutDir,
			"self_transfer_push")
		c.So(ioutil.WriteFile(testFile, testFileContent, 0o600), ShouldBeNil)

		trans := &model.Transfer{
			RuleID:     ctx.ClientPush.ID,
			IsServer:   false,
			AgentID:    ctx.Server.ID,
			AccountID:  ctx.LocAccount.ID,
			LocalPath:  "self_transfer_push",
			RemotePath: "self_transfer_push",
			Start:      time.Now(),
		}
		c.So(ctx.DB.Insert(trans).Run(), ShouldBeNil)

		ctx.Trans = trans
		return
	}

	testFile := filepath.Join(ctx.Server.Root, ctx.Server.LocalOutDir, "self_transfer_pull")
	c.So(ioutil.WriteFile(testFile, testFileContent, 0o600), ShouldBeNil)

	trans := &model.Transfer{
		RuleID:     ctx.ClientPull.ID,
		IsServer:   false,
		AgentID:    ctx.Server.ID,
		AccountID:  ctx.LocAccount.ID,
		LocalPath:  "self_transfer_pull",
		RemotePath: "self_transfer_pull",
		Start:      time.Now(),
	}
	c.So(ctx.DB.Insert(trans).Run(), ShouldBeNil)

	ctx.Trans = trans
}

// MakeChan initializes the tasks channels.
func MakeChan(c C) {
	testhelpers.ClientCheckChannel = make(chan string, 10)
	testhelpers.ServerCheckChannel = make(chan string, 10)
	c.Reset(func() {
		if testhelpers.ClientCheckChannel != nil {
			close(testhelpers.ClientCheckChannel)
		}
		if testhelpers.ServerCheckChannel != nil {
			close(testhelpers.ServerCheckChannel)
		}
		testhelpers.ClientCheckChannel = nil
		testhelpers.ServerCheckChannel = nil
	})
}

// RunTransfer executes the transfer in the given context.
func RunTransfer(c C, ctx *Context) {
	pip, err := pipeline.NewClientPipeline(ctx.DB, ctx.Paths, ctx.Trans)
	c.So(err, ShouldBeNil)

	MakeChan(c)
	pip.Run()
	pipeline.WaitEndClientTransfer(c, pip)
	testhelpers.ClientCheckChannel <- "CLIENT TRANSFER END"
}

// CheckTransfersOK checks whether both the server & client transfers finished
// correctly.
func CheckTransfersOK(c C, ctx *Context) {
	c.Convey("Then the transfers should be over", func(c C) {
		var transfers model.Transfers
		c.So(ctx.DB.Select(&transfers).Run(), ShouldBeNil)
		c.So(transfers, ShouldBeEmpty)

		var results model.HistoryEntries
		c.So(ctx.DB.Select(&results).OrderBy("id", true).Run(), ShouldBeNil)
		c.So(len(results), ShouldEqual, 2)

		c.Convey("Then there should be a client-side history entry", func(c C) {
			cTrans := model.HistoryEntry{
				ID:         ctx.Trans.ID,
				Owner:      database.Owner,
				Protocol:   ctx.Partner.Protocol,
				IsServer:   false,
				Account:    ctx.RemAccount.Login,
				Agent:      ctx.Partner.Name,
				Start:      results[0].Start,
				Stop:       results[0].Stop,
				LocalPath:  ctx.Trans.LocalPath,
				RemotePath: ctx.Trans.RemotePath,
				Status:     types.StatusDone,
				Step:       types.StepNone,
				Error:      types.TransferError{},
				Progress:   uint64(len(testFileContent)),
				TaskNumber: 0,
			}
			if ctx.isPush() {
				cTrans.IsSend = true
				cTrans.Rule = ctx.ClientPush.Name
			} else {
				cTrans.IsSend = false
				cTrans.Rule = ctx.ClientPull.Name
			}
			c.So(results[0], ShouldResemble, cTrans)
		})

		c.Convey("Then there should be a server-side history entry", func(c C) {
			sTrans := model.HistoryEntry{
				ID:         ctx.Trans.ID + 1,
				Owner:      database.Owner,
				Protocol:   ctx.Server.Protocol,
				IsServer:   true,
				Account:    ctx.LocAccount.Login,
				Agent:      ctx.Server.Name,
				Start:      results[1].Start,
				Stop:       results[1].Stop,
				RemotePath: "/" + filepath.Base(ctx.Trans.LocalPath),
				Status:     types.StatusDone,
				Step:       types.StepNone,
				Error:      types.TransferError{},
				Progress:   uint64(len(testFileContent)),
				TaskNumber: 0,
			}
			if ctx.Server.Protocol == "r66" {
				sTrans.RemoteTransferID = fmt.Sprint(ctx.Trans.ID)
			}
			if ctx.isPush() {
				sTrans.IsSend = false
				sTrans.Rule = ctx.ServerPush.Name
				sTrans.LocalPath = filepath.Join(ctx.Server.Root, ctx.Server.LocalInDir,
					filepath.Base(ctx.Trans.LocalPath))
			} else {
				sTrans.IsSend = true
				sTrans.Rule = ctx.ServerPull.Name
				sTrans.LocalPath = filepath.Join(ctx.Server.Root, ctx.Server.LocalOutDir,
					filepath.Base(ctx.Trans.LocalPath))
			}
			c.So(results[1], ShouldResemble, sTrans)
		})
	})

	checkDestFile(c, ctx)
}

func checkDestFile(c C, ctx *Context) {
	c.Convey("Then the file should have been sent entirely", func(c C) {
		path := ctx.Trans.LocalPath
		if ctx.isPush() {
			path = filepath.Join(ctx.Server.Root, ctx.Server.LocalInDir,
				filepath.Base(ctx.Trans.LocalPath))
		}
		content, err := ioutil.ReadFile(path)
		c.So(err, ShouldBeNil)
		c.So(string(content), ShouldEqual, string(testFileContent))
	})
}

func CheckTransfersError(c C, ctx *Context, cTrans, sTrans *model.Transfer) {
	c.Convey("Then the transfers should be in error", func(c C) {
		var transfers model.Transfers
		c.So(ctx.DB.Select(&transfers).OrderBy("id", true).Run(), ShouldBeNil)
		c.So(len(transfers), ShouldEqual, 2)

		c.Convey("Then there should be a client-side transfer in error", func(c C) {
			cTrans.ID = ctx.Trans.ID
			cTrans.Owner = database.Owner
			cTrans.IsServer = false
			cTrans.Status = types.StatusError
			cTrans.RuleID = transfers[0].RuleID
			cTrans.LocalPath = transfers[0].LocalPath
			cTrans.RemotePath = transfers[0].RemotePath
			cTrans.AccountID = ctx.RemAccount.ID
			cTrans.AgentID = ctx.Partner.ID
			cTrans.Start = transfers[0].Start

			c.So(transfers[0], ShouldResemble, *cTrans)
		})

		c.Convey("Then there should be a server-side transfer in error", func(c C) {
			sTrans.ID = ctx.Trans.ID + 1
			sTrans.Owner = database.Owner
			sTrans.IsServer = true
			sTrans.Status = types.StatusError
			sTrans.RuleID = transfers[1].RuleID
			sTrans.LocalPath = transfers[1].LocalPath
			sTrans.RemotePath = transfers[1].RemotePath
			sTrans.AccountID = ctx.LocAccount.ID
			sTrans.AgentID = ctx.Server.ID
			sTrans.Start = transfers[1].Start

			c.So(transfers[1], ShouldResemble, *sTrans)
		})
	})
}
