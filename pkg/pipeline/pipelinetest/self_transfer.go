// Package pipelinetest regroups a series of utility functions and structs for
// quickly instantiating & running transfer pipelines for test purposes.
package pipelinetest

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tasks/taskstest"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/gatewayd"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"
	"github.com/smartystreets/goconvey/convey"
)

// SelfContext is a struct regrouping all the necessary elements to perform
// self-transfer tests, including transfer failure tests.
type SelfContext struct {
	*testData
	*clientData
	*serverData
	*transData
	ClientRule, ServerRule *model.Rule

	service       service.ProtoService
	fail          *model.Task
	protoFeatures *features
}

func initSelfTransfer(c convey.C, proto string, partConf, servConf config.ProtoConfig) *SelfContext {
	feat, ok := protocols[proto]
	c.So(ok, convey.ShouldBeTrue)
	t := initTestData(c)
	port := testhelpers.GetFreePort(c)
	partner, remAcc := makeClientConf(c, t.DB, port, proto, partConf)
	server, locAcc := makeServerConf(c, t.DB, port, t.Paths.GatewayHome, proto, servConf)

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
		transData:     &transData{},
		protoFeatures: &feat,
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
		Filesize:   model.UnknownSize,
		Start:      time.Now(),
	}
	c.So(s.DB.Insert(trans).Run(), convey.ShouldBeNil)

	s.ClientTrans = trans
}

// StartService starts the service associated with the test server defined in
// the SelfContext.
func (s *SelfContext) StartService(c convey.C) {
	s.service = gatewayd.ServiceConstructors[s.Server.Protocol](s.DB, s.Server, s.Logger)
	c.So(s.service.Start(), convey.ShouldBeNil)
	c.Reset(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		c.So(s.service.Stop(ctx), convey.ShouldBeNil)
	})
}

// AddCryptos adds the given cryptos to the test database.
func (s *SelfContext) AddCryptos(c convey.C, certs ...model.Crypto) {
	for i := range certs {
		c.So(s.DB.Insert(&certs[i]).Run(), convey.ShouldBeNil)
	}
}

// AddClientPreTaskError purposefully adds an error in the client's transfer
// rule's pre-tasks to test error handling.
func (s *SelfContext) AddClientPreTaskError(c convey.C) {
	c.So(s.fail, convey.ShouldBeNil)
	s.fail = &model.Task{
		RuleID: s.ClientRule.ID,
		Chain:  model.ChainPre,
		Rank:   1,
		Type:   taskstest.TaskErr,
	}
	c.So(s.DB.Insert(s.fail).Run(), convey.ShouldBeNil)
}

// AddClientPostTaskError purposefully adds an error in the client's transfer
// rule's post-tasks to test error handling.
func (s *SelfContext) AddClientPostTaskError(c convey.C) {
	c.So(s.fail, convey.ShouldBeNil)
	s.fail = &model.Task{
		RuleID: s.ClientRule.ID,
		Chain:  model.ChainPost,
		Rank:   1,
		Type:   taskstest.TaskErr,
	}
	c.So(s.DB.Insert(s.fail).Run(), convey.ShouldBeNil)
}

// AddServerPreTaskError purposefully adds an error in the server's transfer
// rule's pre-tasks to test error handling.
func (s *SelfContext) AddServerPreTaskError(c convey.C) {
	c.So(s.fail, convey.ShouldBeNil)
	s.fail = &model.Task{
		RuleID: s.ServerRule.ID,
		Chain:  model.ChainPre,
		Rank:   1,
		Type:   taskstest.TaskErr,
	}
	c.So(s.DB.Insert(s.fail).Run(), convey.ShouldBeNil)
}

// AddServerPostTaskError purposefully adds an error in the server's transfer
// rule's post-tasks to test error handling.
func (s *SelfContext) AddServerPostTaskError(c convey.C) {
	c.So(s.fail, convey.ShouldBeNil)
	s.fail = &model.Task{
		RuleID: s.ServerRule.ID,
		Chain:  model.ChainPost,
		Rank:   1,
		Type:   taskstest.TaskErr,
	}
	c.So(s.DB.Insert(s.fail).Run(), convey.ShouldBeNil)
}

// RunTransfer executes the test self-transfer in its entirety.
func (s *SelfContext) RunTransfer(c convey.C) {
	pip, err := pipeline.NewClientPipeline(s.DB, s.ClientTrans)
	c.So(err, convey.ShouldBeNil)

	pip.Run()
	s.TasksChecker.WaitClientDone()
	s.TasksChecker.WaitServerDone()
	s.shouldNotBeInLists()
}

func (s *SelfContext) resetTransfer(c convey.C) {
	c.So(s.DB.DeleteAll(&model.Task{}).Where("type=?", taskstest.TaskErr).
		Run(), convey.ShouldBeNil)
	s.ClientTrans.Status = types.StatusPlanned
	c.So(s.DB.Update(s.ClientTrans).Cols("status").Run(), convey.ShouldBeNil)
	s.TasksChecker.Retry()
}

// TestRetry can be called to test
func (s *SelfContext) TestRetry(c convey.C, checkRemainingTasks ...func(c convey.C)) {
	c.Convey("When retrying the transfer", func(c convey.C) {
		s.resetTransfer(c)
		s.RunTransfer(c)

		c.Convey("Then it should have executed all the tasks in order", func(c convey.C) {
			for _, f := range checkRemainingTasks {
				f(c)
			}

			s.CheckEndTransferOK(c)
		})
	})
}

func (s *SelfContext) checkClientTransferOK(c convey.C, actual *model.HistoryEntry) {
	c.Convey("Then there should be a client-side history entry", func(c convey.C) {
		expected := &model.HistoryEntry{
			ID:         s.ClientTrans.ID,
			Owner:      s.DB.Conf.GatewayName,
			Protocol:   s.Partner.Protocol,
			Rule:       s.ClientRule.Name,
			IsServer:   false,
			IsSend:     s.ClientRule.IsSend,
			Account:    s.RemAccount.Login,
			Agent:      s.Partner.Name,
			Start:      actual.Start,
			Stop:       actual.Stop,
			LocalPath:  s.ClientTrans.LocalPath,
			RemotePath: s.ClientTrans.RemotePath,
			Filesize:   TestFileSize,
			Status:     types.StatusDone,
			Step:       types.StepNone,
			Error:      types.TransferError{},
			Progress:   uint64(len(s.transData.fileContent)),
			TaskNumber: 0,
		}
		c.So(*actual, convey.ShouldResemble, *expected)
	})
}

// CheckClientTransferOK checks if the client transfer history entry has
// succeeded as expected.
func (s *SelfContext) CheckClientTransferOK(c convey.C) {
	var actual model.HistoryEntry
	c.So(s.DB.Get(&actual, "id=?", s.ClientTrans.ID).Run(), convey.ShouldBeNil)
	s.checkClientTransferOK(c, &actual)
}

func (s *SelfContext) checkServerTransferOK(c convey.C, actual *model.HistoryEntry) {
	c.Convey("Then there should be a server-side history entry", func(c convey.C) {
		expected := &model.HistoryEntry{
			ID:         actual.ID,
			Owner:      s.DB.Conf.GatewayName,
			Protocol:   s.Server.Protocol,
			IsServer:   true,
			IsSend:     s.ServerRule.IsSend,
			Rule:       s.ServerRule.Name,
			Account:    s.LocAccount.Login,
			Agent:      s.Server.Name,
			Start:      actual.Start,
			Stop:       actual.Stop,
			RemotePath: "/" + filepath.Base(s.ClientTrans.LocalPath),
			Filesize:   TestFileSize,
			Status:     types.StatusDone,
			Step:       types.StepNone,
			Error:      types.TransferError{},
			Progress:   uint64(len(s.transData.fileContent)),
			TaskNumber: 0,
		}
		if s.protoFeatures.transID {
			expected.RemoteTransferID = fmt.Sprint(s.ClientTrans.ID)
		}
		if s.ServerRule.IsSend {
			expected.LocalPath = filepath.Join(s.Server.Root, s.Server.LocalOutDir,
				filepath.Base(s.ClientTrans.LocalPath))
		} else {
			expected.LocalPath = filepath.Join(s.Server.Root, s.Server.LocalInDir,
				filepath.Base(s.ClientTrans.LocalPath))
		}
		c.So(*actual, convey.ShouldResemble, *expected)
	})
}

// CheckServerTransferOK checks if the server transfer history entry has
// succeeded as expected.
func (s *SelfContext) CheckServerTransferOK(c convey.C) {
	var actual model.HistoryEntry
	c.So(s.DB.Get(&actual, "id=?", s.ClientTrans.ID+1).Run(), convey.ShouldBeNil)
	s.checkServerTransferOK(c, &actual)
}

// CheckEndTransferOK checks whether both the server & client test transfers
// finished correctly.
func (s *SelfContext) CheckEndTransferOK(c convey.C) {
	c.Convey("Then the transfers should be over", func(c convey.C) {
		var results model.HistoryEntries
		c.So(s.DB.Select(&results).OrderBy("id", true).Run(), convey.ShouldBeNil)
		c.So(len(results), convey.ShouldEqual, 2)

		s.checkClientTransferOK(c, &results[0])
		s.checkServerTransferOK(c, &results[1])
	})

	s.CheckDestFile(c)
}

// CheckDestFile checks if the transfer's destination file does exist, and if
// its content is as expected.
func (s *SelfContext) CheckDestFile(c convey.C) {
	c.Convey("Then the file should have been sent entirely", func(c convey.C) {
		path := s.ClientTrans.LocalPath
		if s.ClientRule.IsSend {
			path = filepath.Join(s.Server.Root, s.Server.LocalInDir,
				filepath.Base(s.ClientTrans.LocalPath))
		}
		content, err := ioutil.ReadFile(path)
		c.So(err, convey.ShouldBeNil)
		c.So(len(content), convey.ShouldEqual, TestFileSize)
		c.So(content[:9], convey.ShouldResemble, s.fileContent[:9])
		c.So(content, convey.ShouldResemble, s.fileContent)
	})
}

//nolint:dupl
// CheckClientTransferError takes asserts that the client transfer should have
// failed like the given expected one. The expected entry must specify the step,
// filesize (for the receiver), progress, task. The rest of the transfer entry's
// attribute will be deduced automatically.
func (s *SelfContext) CheckClientTransferError(c convey.C, errCode types.TransferErrorCode,
	errMsg string, steps ...types.TransferStep) {

	var actual model.Transfer
	c.So(s.DB.Get(&actual, "id=?", s.ClientTrans.ID).Run(), convey.ShouldBeNil)

	var stepsStr []string
	for _, s := range steps {
		stepsStr = append(stepsStr, s.String())
	}

	c.Convey("Then there should be a client-side transfer in error", func(c convey.C) {
		c.So(actual.ID, convey.ShouldEqual, 1)
		c.So(actual.Owner, convey.ShouldEqual, s.DB.Conf.GatewayName)
		c.So(actual.IsServer, convey.ShouldBeFalse)
		c.So(actual.Status, convey.ShouldEqual, types.StatusError)
		c.So(actual.RuleID, convey.ShouldEqual, s.ClientRule.ID)
		c.So(actual.AccountID, convey.ShouldEqual, s.RemAccount.ID)
		c.So(actual.AgentID, convey.ShouldEqual, s.Partner.ID)
		c.So(actual.Status, convey.ShouldEqual, types.StatusError)

		err := types.TransferError{Code: errCode, Details: errMsg}
		c.So(actual.Error, convey.ShouldResemble, err)
		c.So(actual.Filesize, testhelpers.ShouldBeOneOf, model.UnknownSize, TestFileSize)
		c.So(actual.Progress, convey.ShouldBeBetweenOrEqual, 0, TestFileSize)
		c.So(actual.Step.String(), testhelpers.ShouldBeOneOf, stepsStr)

		if actual.Step == types.StepPreTasks || actual.Step == types.StepPostTasks {
			c.So(actual.TaskNumber, convey.ShouldEqual, 1)
		} else {
			c.So(actual.TaskNumber, convey.ShouldEqual, 0)
		}
	})
}

//nolint:dupl
// CheckServerTransferError takes asserts that the server transfer should have
// failed like the given expected one. The expected entry must specify the step,
// filesize (for the receiver), progress, task. The rest of the transfer entry's
// attribute will be deduced automatically.
func (s *SelfContext) CheckServerTransferError(c convey.C, errCode types.TransferErrorCode,
	errMsg string, steps ...types.TransferStep) {

	var actual model.Transfer
	c.So(s.DB.Get(&actual, "id=?", s.ClientTrans.ID+1).Run(), convey.ShouldBeNil)

	var stepsStr []string
	for _, s := range steps {
		stepsStr = append(stepsStr, s.String())
	}

	c.Convey("Then there should be a server-side transfer in error", func(c convey.C) {
		c.So(actual.ID, convey.ShouldEqual, 2)
		c.So(actual.Owner, convey.ShouldEqual, s.DB.Conf.GatewayName)
		c.So(actual.IsServer, convey.ShouldBeTrue)
		c.So(actual.Status, convey.ShouldEqual, types.StatusError)
		c.So(actual.RuleID, convey.ShouldEqual, s.ServerRule.ID)
		c.So(actual.AccountID, convey.ShouldEqual, s.LocAccount.ID)
		c.So(actual.AgentID, convey.ShouldEqual, s.Server.ID)
		c.So(actual.Status, convey.ShouldEqual, types.StatusError)

		err := types.TransferError{Code: errCode, Details: errMsg}
		c.So(actual.Error, convey.ShouldResemble, err)
		c.So(actual.Filesize, testhelpers.ShouldBeOneOf, model.UnknownSize, TestFileSize)
		c.So(actual.Progress, convey.ShouldBeBetweenOrEqual, 0, TestFileSize)
		c.So(actual.Step.String(), testhelpers.ShouldBeOneOf, stepsStr)

		if actual.Step == types.StepPreTasks || actual.Step == types.StepPostTasks {
			c.So(actual.TaskNumber, convey.ShouldEqual, 1)
		} else {
			c.So(actual.TaskNumber, convey.ShouldEqual, 0)
		}
	})
}

func (s *SelfContext) shouldNotBeInLists() {
	ok := pipeline.ClientTransfers.Exists(s.ClientTrans.ID)
	convey.So(ok, convey.ShouldBeFalse)
	ok = s.service.ManageTransfers().Exists(s.ClientTrans.ID + 1)
	convey.So(ok, convey.ShouldBeFalse)
}
