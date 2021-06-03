package pipelinetest

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/gatewayd"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"
	"github.com/smartystreets/goconvey/convey"
)

type SelfContext struct {
	*testData
	*clientData
	*serverData
	*transData
	ClientRule, ServerRule *model.Rule

	fail                *model.Task
	cliChain, servChain model.Chain
	cliIndex, servIndex int
}

func initSelfTransfer(c convey.C, proto string, partConf, servConf config.ProtoConfig) *SelfContext {
	t := initTestData(c)
	port := testhelpers.GetFreePort(c)
	partner, remAcc := makeClientConf(c, t.DB, port, proto, partConf)
	server, locAcc := makeServerConf(c, t.DB, port, t.Paths.GatewayHome, proto, servConf)
	setTestVar()

	return &SelfContext{
		testData: t,
		clientData: &clientData{
			Partner:    partner,
			RemAccount: remAcc,
		},
		serverData: &serverData{
			Server:     server,
			LocAccount: locAcc,
		},
		transData: &transData{},
	}
}

// InitSelfPushTransfer creates a database and fills it with all the elements
// necessary for a push self-transfer test of the given protocol. It then returns
// all these element inside a SelfContext.
func InitSelfPushTransfer(c convey.C, proto string, partConf, servConf config.ProtoConfig) *SelfContext {
	ctx := initSelfTransfer(c, proto, partConf, servConf)
	ctx.ClientRule = makeClientPush(c, ctx.DB)
	ctx.ServerRule = makeServerPush(c, ctx.DB)
	ctx.addPushTransfer(c)
	return ctx
}

// InitSelfPullTransfer creates a database and fills it with all the elements
// necessary for a pull self-transfer test of the given protocol. It then returns
// all these element inside a SelfContext.
func InitSelfPullTransfer(c convey.C, proto string, partConf, servConf config.ProtoConfig) *SelfContext {
	ctx := initSelfTransfer(c, proto, partConf, servConf)
	ctx.ClientRule = makeClientPull(c, ctx.DB)
	ctx.ServerRule = makeServerPull(c, ctx.DB)
	ctx.addPullTransfer(c)
	return ctx
}

func (s *SelfContext) addPushTransfer(c convey.C) {
	testDir := filepath.Join(s.Paths.GatewayHome, s.Paths.DefaultOutDir)
	s.fileContent = AddSourceFile(c, testDir, "self_transfer_push")

	trans := &model.Transfer{
		RuleID:     s.ClientRule.ID,
		IsServer:   false,
		AgentID:    s.Server.ID,
		AccountID:  s.LocAccount.ID,
		LocalPath:  "self_transfer_push",
		RemotePath: "self_transfer_push",
		Start:      time.Now(),
	}
	c.So(s.DB.Insert(trans).Run(), convey.ShouldBeNil)

	s.ClientTrans = trans
	return
}

func (s *SelfContext) addPullTransfer(c convey.C) {
	testDir := filepath.Join(s.Server.Root, s.Server.LocalOutDir)
	s.fileContent = AddSourceFile(c, testDir, "self_transfer_pull")

	trans := &model.Transfer{
		RuleID:     s.ClientRule.ID,
		IsServer:   false,
		AgentID:    s.Server.ID,
		AccountID:  s.LocAccount.ID,
		LocalPath:  "self_transfer_pull",
		RemotePath: "self_transfer_pull",
		Start:      time.Now(),
	}
	c.So(s.DB.Insert(trans).Run(), convey.ShouldBeNil)

	s.ClientTrans = trans
}

func (s *SelfContext) StartService(c convey.C) {
	serv := gatewayd.ServiceConstructors[s.Server.Protocol](s.DB, s.Server, s.Logger)
	c.So(serv.Start(), convey.ShouldBeNil)
	c.Reset(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		c.So(serv.Stop(ctx), convey.ShouldBeNil)
	})
}

func (s *SelfContext) AddCerts(c convey.C, certs ...model.Cert) {
	for _, cert := range certs {
		c.So(s.DB.Insert(&cert).Run(), convey.ShouldBeNil)
	}
}

func (s *SelfContext) AddClientPreTaskError(c convey.C) {
	c.So(s.fail, convey.ShouldBeNil)
	s.fail = &model.Task{
		RuleID: s.ClientRule.ID,
		Chain:  model.ChainPre,
		Rank:   1,
		Type:   testhelpers.ClientErr,
		Args:   json.RawMessage(`{"msg":"PRE-TASKS[1]"}`),
	}
	c.So(s.DB.Insert(s.fail).Run(), convey.ShouldBeNil)
}

func (s *SelfContext) AddClientPostTaskError(c convey.C) {
	c.So(s.fail, convey.ShouldBeNil)
	s.fail = &model.Task{
		RuleID: s.ClientRule.ID,
		Chain:  model.ChainPost,
		Rank:   1,
		Type:   testhelpers.ClientErr,
		Args:   json.RawMessage(`{"msg":"POST-TASKS[1]"}`),
	}
	c.So(s.DB.Insert(s.fail).Run(), convey.ShouldBeNil)
}

func (s *SelfContext) AddServerPreTaskError(c convey.C) {
	c.So(s.fail, convey.ShouldBeNil)
	s.fail = &model.Task{
		RuleID: s.ServerRule.ID,
		Chain:  model.ChainPre,
		Rank:   1,
		Type:   testhelpers.ServerErr,
		Args:   json.RawMessage(`{"msg":"PRE-TASKS[1]"}`),
	}
	c.So(s.DB.Insert(s.fail).Run(), convey.ShouldBeNil)
}

func (s *SelfContext) AddServerPostTaskError(c convey.C) {
	c.So(s.fail, convey.ShouldBeNil)
	s.fail = &model.Task{
		RuleID: s.ServerRule.ID,
		Chain:  model.ChainPost,
		Rank:   1,
		Type:   testhelpers.ServerErr,
		Args:   json.RawMessage(`{"msg":"POST-TASKS[1]"}`),
	}
	c.So(s.DB.Insert(s.fail).Run(), convey.ShouldBeNil)
}

func (s *SelfContext) RunTransfer(c convey.C) {
	pip, err := pipeline.NewClientPipeline(s.DB, s.ClientTrans)
	c.So(err, convey.ShouldBeNil)

	pip.Run()
}

func (s *SelfContext) ResetTransfer(c convey.C) {
	c.So(s.DB.DeleteAll(&model.Task{}).Where("type=? OR type=?", testhelpers.ClientErr, testhelpers.ServerErr).
		Run(), convey.ShouldBeNil)
	s.ClientTrans.Status = types.StatusPlanned
	c.So(s.DB.Update(s.ClientTrans).Cols("status").Run(), convey.ShouldBeNil)
}

func (s *SelfContext) TestRetry(c convey.C, checkRemainingTasks ...func(c convey.C)) {
	c.Convey("When retrying the transfer", func(c convey.C) {
		s.ResetTransfer(c)
		s.RunTransfer(c)

		c.Convey("Then it should have executed all the tasks in order", func(c convey.C) {
			for _, f := range checkRemainingTasks {
				f(c)
			}

			s.CheckTransfersOK(c)
		})
	})
}

// CheckTransfersOK checks whether both the server & client transfers finished
// correctly.
func (s *SelfContext) CheckTransfersOK(c convey.C) {
	s.shouldBeEndTransfer(c)

	c.Convey("Then the transfers should be over", func(c convey.C) {
		var results model.HistoryEntries
		c.So(s.DB.Select(&results).OrderBy("id", true).Run(), convey.ShouldBeNil)
		c.So(len(results), convey.ShouldEqual, 2)

		c.Convey("Then there should be a client-side history entry", func(c convey.C) {
			cTrans := model.HistoryEntry{
				ID:         s.ClientTrans.ID,
				Owner:      s.DB.Conf.GatewayName,
				Protocol:   s.Partner.Protocol,
				Rule:       s.ClientRule.Name,
				IsServer:   false,
				IsSend:     s.ClientRule.IsSend,
				Account:    s.RemAccount.Login,
				Agent:      s.Partner.Name,
				Start:      results[0].Start,
				Stop:       results[0].Stop,
				LocalPath:  s.ClientTrans.LocalPath,
				RemotePath: s.ClientTrans.RemotePath,
				Status:     types.StatusDone,
				Step:       types.StepNone,
				Error:      types.TransferError{},
				Progress:   uint64(len(s.fileContent)),
				TaskNumber: 0,
			}
			c.So(results[0], convey.ShouldResemble, cTrans)
		})

		c.Convey("Then there should be a server-side history entry", func(c convey.C) {
			sTrans := model.HistoryEntry{
				ID:         results[1].ID,
				Owner:      s.DB.Conf.GatewayName,
				Protocol:   s.Server.Protocol,
				IsServer:   true,
				IsSend:     s.ServerRule.IsSend,
				Rule:       s.ServerRule.Name,
				Account:    s.LocAccount.Login,
				Agent:      s.Server.Name,
				Start:      results[1].Start,
				Stop:       results[1].Stop,
				RemotePath: "/" + filepath.Base(s.ClientTrans.LocalPath),
				Status:     types.StatusDone,
				Step:       types.StepNone,
				Error:      types.TransferError{},
				Progress:   uint64(len(s.fileContent)),
				TaskNumber: 0,
			}
			if s.Server.Protocol == "r66" {
				sTrans.RemoteTransferID = fmt.Sprint(s.ClientTrans.ID)
			}
			if s.ServerRule.IsSend {
				sTrans.LocalPath = filepath.Join(s.Server.Root, s.Server.LocalOutDir,
					filepath.Base(s.ClientTrans.LocalPath))
			} else {
				sTrans.LocalPath = filepath.Join(s.Server.Root, s.Server.LocalInDir,
					filepath.Base(s.ClientTrans.LocalPath))
			}
			c.So(results[1], convey.ShouldResemble, sTrans)
		})
	})

	s.checkDestFile(c)
}

func (s *SelfContext) checkDestFile(c convey.C) {
	c.Convey("Then the file should have been sent entirely", func(c convey.C) {
		path := s.ClientTrans.LocalPath
		if s.ClientRule.IsSend {
			path = filepath.Join(s.Server.Root, s.Server.LocalInDir,
				filepath.Base(s.ClientTrans.LocalPath))
		}
		content, err := ioutil.ReadFile(path)
		c.So(err, convey.ShouldBeNil)
		c.So(content, convey.ShouldHaveLength, TestFileSize)
		c.So(content[:9], convey.ShouldResemble, s.fileContent[:9])
		c.So(content, convey.ShouldResemble, s.fileContent)
	})
}

func (s *SelfContext) CheckTransfersError(c convey.C, cTrans, sTrans *model.Transfer) {
	s.shouldBeErrorTasks(c)
	s.shouldBeEndTransfer(c)

	c.Convey("Then the transfers should be in error", func(c convey.C) {
		var transfers model.Transfers
		c.So(s.DB.Select(&transfers).OrderBy("id", true).Run(), convey.ShouldBeNil)
		c.So(len(transfers), convey.ShouldEqual, 2)

		c.Convey("Then there should be a client-side transfer in error", func(c convey.C) {
			cTrans.ID = s.ClientTrans.ID
			cTrans.Owner = s.DB.Conf.GatewayName
			cTrans.IsServer = false
			cTrans.Status = types.StatusError
			cTrans.RuleID = transfers[0].RuleID
			cTrans.LocalPath = transfers[0].LocalPath
			cTrans.RemotePath = transfers[0].RemotePath
			cTrans.AccountID = s.RemAccount.ID
			cTrans.AgentID = s.Partner.ID
			cTrans.Start = transfers[0].Start

			c.So(transfers[0], convey.ShouldResemble, *cTrans)
		})

		c.Convey("Then there should be a server-side transfer in error", func(c convey.C) {
			sTrans.ID = s.ClientTrans.ID + 1
			sTrans.Owner = s.DB.Conf.GatewayName
			sTrans.IsServer = true
			sTrans.Status = types.StatusError
			sTrans.RuleID = transfers[1].RuleID
			sTrans.LocalPath = transfers[1].LocalPath
			sTrans.RemotePath = transfers[1].RemotePath
			sTrans.AccountID = s.LocAccount.ID
			sTrans.AgentID = s.Server.ID
			sTrans.Start = transfers[1].Start
			if s.Server.Protocol == "r66" {
				sTrans.RemoteTransferID = fmt.Sprint(s.ClientTrans.ID)
			}

			c.So(transfers[1], convey.ShouldResemble, *sTrans)
		})
	})
}
