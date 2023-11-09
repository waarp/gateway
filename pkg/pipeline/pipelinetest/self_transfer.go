// Package pipelinetest regroups a series of utility functions and structs for
// quickly instantiating & running transfer pipelines for test purposes.
package pipelinetest

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/proto"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks/taskstest"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

// SelfContext is a struct regrouping all the necessary elements to perform
// self-transfer tests, including transfer failure tests.
type SelfContext struct {
	*testData
	*clientData
	*serverData
	*transData

	constr        serviceConstructor
	service       proto.Service
	fail          *model.Task
	protoFeatures *features
}

func initSelfTransfer(c convey.C, protocol string, constr serviceConstructor,
	partConf, servConf config.ProtoConfig,
) *SelfContext {
	feat, ok := protocols[protocol]
	c.So(ok, convey.ShouldBeTrue)
	t := initTestData(c)
	port := testhelpers.GetFreePort(c)
	partner, remAcc := makeClientConf(c, t.DB, port, protocol, partConf)
	server, locAcc := makeServerConf(c, t, port, protocol, servConf)

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
		transData: &transData{
			transferInfo: map[string]interface{}{},
			// fileInfo:     map[string]interface{}{},
		},
		protoFeatures: &feat,
		constr:        constr,
	}
}

// InitSelfPushTransfer creates a database and fills it with all the elements
// necessary for a push self-transfer test of the given protocol. It then returns
// all these element inside a SelfContext.
func InitSelfPushTransfer(c convey.C, protocol string, constr serviceConstructor,
	partConf, servConf config.ProtoConfig,
) *SelfContext {
	ctx := initSelfTransfer(c, protocol, constr, partConf, servConf)
	ctx.ClientRule = makeClientPush(c, ctx.DB, protocol)
	ctx.ServerRule = makeServerPush(c, ctx.DB)
	ctx.addPushTransfer(c)

	return ctx
}

// InitSelfPullTransfer creates a database and fills it with all the elements
// necessary for a pull self-transfer test of the given protocol. It then returns
// all these element inside a SelfContext.
func InitSelfPullTransfer(c convey.C, protocol string, constr serviceConstructor,
	partConf, servConf config.ProtoConfig,
) *SelfContext {
	ctx := initSelfTransfer(c, protocol, constr, partConf, servConf)
	ctx.ClientRule = makeClientPull(c, ctx.DB, protocol)
	ctx.ServerRule = makeServerPull(c, ctx.DB)
	ctx.addPullTransfer(c)

	return ctx
}

//nolint:dupl // factorizing would hurt readability
func (s *SelfContext) addPushTransfer(c convey.C) {
	filePath := mkURL(s.Paths.GatewayHome, s.ClientRule.LocalDir,
		"sub_dir", "self_transfer_push")
	s.fileContent = AddSourceFile(c, s.FS, filePath)

	trans := &model.Transfer{
		RuleID:          s.ClientRule.ID,
		RemoteAccountID: utils.NewNullInt64(s.RemAccount.ID),
		SrcFilename:     "sub_dir/self_transfer_push",
		Start:           time.Now(),
	}
	c.So(s.DB.Insert(trans).Run(), convey.ShouldBeNil)

	s.ClientTrans = trans
}

//nolint:dupl // factorizing would hurt readability
func (s *SelfContext) addPullTransfer(c convey.C) {
	filePath := mkURL(s.Paths.GatewayHome, s.Server.RootDir,
		s.ServerRule.LocalDir, s.getClientRemoteDir(), "sub_dir", "self_transfer_pull")
	s.fileContent = AddSourceFile(c, s.FS, filePath)

	trans := &model.Transfer{
		RuleID:          s.ClientRule.ID,
		RemoteAccountID: utils.NewNullInt64(s.RemAccount.ID),
		SrcFilename:     "sub_dir/self_transfer_pull",
		Filesize:        model.UnknownSize,
		Start:           time.Now(),
	}
	c.So(s.DB.Insert(trans).Run(), convey.ShouldBeNil)

	s.ClientTrans = trans
}

// StartService starts the service associated with the test server defined in
// the SelfContext.
func (s *SelfContext) StartService(c convey.C) {
	logger := conf.GetLogger(fmt.Sprintf("test_%s_server", s.Server.Protocol))
	s.service = s.constr(s.DB, logger)
	c.So(s.service.Start(s.Server), convey.ShouldBeNil)
	c.Reset(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		c.So(s.service.Stop(ctx), convey.ShouldBeNil)
	})
}

// Service returns the context's service.
func (s *SelfContext) Service() proto.Service { return s.service }

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

// DataErrorOffset defines the offset at which simulated data errors should occur.
const DataErrorOffset = TestFileSize / 2

func (s *SelfContext) AddClientDataError(_ convey.C) {
	if s.ClientRule.IsSend {
		pipeline.Tester.AddErrorAt(pipeline.DataRead, DataErrorOffset)
	} else {
		pipeline.Tester.AddErrorAt(pipeline.DataWrite, DataErrorOffset)
	}
}

func (s *SelfContext) AddServerDataError(_ convey.C) {
	if s.ClientRule.IsSend {
		pipeline.Tester.AddErrorAt(pipeline.DataWrite, DataErrorOffset)
	} else {
		pipeline.Tester.AddErrorAt(pipeline.DataRead, DataErrorOffset)
	}
}

// RunTransfer executes the test self-transfer in its entirety.
func (s *SelfContext) RunTransfer(c convey.C, willFail bool) {
	pip, err := pipeline.NewClientPipeline(s.DB, s.ClientTrans)
	c.So(err, convey.ShouldBeNil)

	if tErr := pip.Run(); !willFail {
		convey.So(tErr, convey.ShouldBeNil)
	}

	pipeline.Tester.WaitClientDone()
	pipeline.Tester.WaitServerDone()
	s.waitForListDeletion()
}

func (s *SelfContext) resetTransfer(c convey.C) {
	pipeline.Tester.Retry()

	c.So(s.DB.DeleteAll(&model.Task{}).Where("type=?", taskstest.TaskErr).
		Run(), convey.ShouldBeNil)

	s.ClientTrans.Status = types.StatusPlanned
	c.So(s.DB.Update(s.ClientTrans).Run(), convey.ShouldBeNil)
}

// TestRetry can be called to test a transfer retry.
func (s *SelfContext) TestRetry(c convey.C, checkRemainingTasks ...func(c convey.C)) {
	c.Convey("When retrying the transfer", func(c convey.C) {
		s.resetTransfer(c)
		s.RunTransfer(c, false)

		c.Convey("Then it should have executed all the tasks in order", func(c convey.C) {
			for _, f := range checkRemainingTasks {
				f(c)
			}

			s.CheckEndTransferOK(c)
		})
	})
}

// CheckClientTransferOK checks if the client transfer history entry has
// succeeded as expected.
func (s *SelfContext) CheckClientTransferOK(c convey.C) {
	var actual model.HistoryEntry

	c.So(s.DB.Get(&actual, "id=?", s.ClientTrans.ID).Run(), convey.ShouldBeNil)
	s.checkClientTransferOK(c, s.transData, s.DB, &actual)
}

func (s *SelfContext) getClientRemoteDir() string {
	ruleDir := s.ClientRule.RemoteDir

	if !s.protoFeatures.ruleName {
		ruleDir = strings.TrimPrefix(ruleDir, s.ServerRule.Path+"/")
	}

	return ruleDir
}

func (s *SelfContext) checkServerTransferOK(c convey.C, actual *model.HistoryEntry) {
	remoteID := s.transData.ClientTrans.RemoteTransferID
	if !s.protoFeatures.transID {
		remoteID = actual.RemoteTransferID
	}

	progress := int64(len(s.fileContent))
	filename := path.Join(s.getClientRemoteDir(), s.ClientTrans.SrcFilename)

	s.serverData.checkServerTransferOK(c, remoteID, filename, progress, s.testData,
		actual, s.transData)
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
		var trans model.Transfers

		c.So(s.DB.Select(&trans).Run(), convey.ShouldBeNil)
		c.So(trans, convey.ShouldBeEmpty)

		var results model.HistoryEntries

		c.So(s.DB.Select(&results).OrderBy("id", true).Run(), convey.ShouldBeNil)
		c.So(results, convey.ShouldHaveLength, 2) //nolint:gomnd // necessary here

		s.checkClientTransferOK(c, s.transData, s.DB, results[0])
		s.checkServerTransferOK(c, results[1])
	})

	s.CheckDestFile(c)
}

// CheckDestFile checks if the transfer's destination file does exist, and if
// its content is as expected.
func (s *SelfContext) CheckDestFile(c convey.C) {
	c.Convey("Then the file should have been sent entirely", func(c convey.C) {
		fullPath := &s.ClientTrans.LocalPath

		if s.ClientRule.IsSend {
			fullPathStr := path.Join(s.Paths.GatewayHome, s.Server.RootDir,
				s.ServerRule.LocalDir, s.getClientRemoteDir(), s.ClientTrans.SrcFilename)
			fullPath = mkURL(fullPathStr)
		}

		content, err := fs.ReadFile(s.FS, fullPath)

		c.So(err, convey.ShouldBeNil)
		c.So(len(content), convey.ShouldEqual, TestFileSize)
		c.So(content[:9], convey.ShouldResemble, s.fileContent[:9])
		c.So(content, convey.ShouldResemble, s.fileContent)
	})
}

// CheckClientTransferError takes asserts that the client transfer should have
// failed like the given expected one. The expected entry must specify the step,
// filesize (for the receiver), progress, task. The rest of the transfer entry's
// attribute will be deduced automatically.
//
//nolint:dupl // factorizing would hurt readability
func (s *SelfContext) CheckClientTransferError(c convey.C, errCode types.TransferErrorCode,
	errMsg string, steps ...types.TransferStep,
) {
	actual := s.getTransfer(c, s.ClientTrans.ID)

	var stepsStr []string
	for _, s := range steps {
		stepsStr = append(stepsStr, s.String())
	}

	c.Convey("Then there should be a client-side transfer in error", func(c convey.C) {
		c.So(actual.ID, convey.ShouldEqual, s.ClientTrans.ID)
		c.So(actual.RemoteTransferID, convey.ShouldNotBeBlank)
		c.So(actual.Owner, convey.ShouldEqual, conf.GlobalConfig.GatewayName)
		c.So(actual.Status, convey.ShouldEqual, types.StatusError)
		c.So(actual.RuleID, convey.ShouldEqual, s.ClientRule.ID)
		c.So(actual.RemoteAccountID.Int64, convey.ShouldEqual, s.RemAccount.ID)
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

// CheckServerTransferError takes asserts that the server transfer should have
// failed like the given expected one. The expected entry must specify the step,
// filesize (for the receiver), progress, task. The rest of the transfer entry's
// attribute will be deduced automatically.
//
//nolint:dupl // factorizing would hurt readability
func (s *SelfContext) CheckServerTransferError(c convey.C, errCode types.TransferErrorCode,
	errMsg string, steps ...types.TransferStep,
) {
	id := s.ClientTrans.ID + 1
	actual := s.getTransfer(c, id)

	var stepsStr []string
	for _, s := range steps {
		stepsStr = append(stepsStr, s.String())
	}

	c.Convey("Then there should be a server-side transfer in error", func(c convey.C) {
		c.So(actual.ID, convey.ShouldEqual, id)
		c.So(actual.Owner, convey.ShouldEqual, conf.GlobalConfig.GatewayName)
		c.So(actual.Status, convey.ShouldEqual, types.StatusError)
		c.So(actual.RuleID, convey.ShouldEqual, s.ServerRule.ID)
		c.So(actual.LocalAccountID.Int64, convey.ShouldEqual, s.LocAccount.ID)
		c.So(actual.Status, convey.ShouldEqual, types.StatusError)

		err := types.TransferError{Code: errCode, Details: errMsg}
		c.So(actual.Error, convey.ShouldResemble, err)
		c.So(actual.Filesize, testhelpers.ShouldBeOneOf, model.UnknownSize, TestFileSize)
		c.So(actual.Progress, convey.ShouldBeBetweenOrEqual, 0, TestFileSize)
		c.So(actual.Step.String(), testhelpers.ShouldBeOneOf, stepsStr)

		if s.protoFeatures.transID {
			c.So(actual.RemoteTransferID, convey.ShouldEqual, s.ClientTrans.RemoteTransferID)
		}

		if actual.Step == types.StepPreTasks || actual.Step == types.StepPostTasks {
			c.So(actual.TaskNumber, convey.ShouldEqual, 1)
		} else {
			c.So(actual.TaskNumber, convey.ShouldEqual, 0)
		}
	})
}

func (s *SelfContext) getTransfer(c convey.C, id int64) *model.Transfer {
	var transfers model.Transfers

	c.So(s.DB.Select(&transfers).Run(), convey.ShouldBeNil)
	c.So(transfers, convey.ShouldNotBeEmpty)

	for i := range transfers {
		if transfers[i].ID == id {
			return transfers[i]
		}
	}

	c.So(transfers, convey.ShouldBeEmpty)

	return nil
}

func (s *SelfContext) waitForListDeletion() {
	timer := time.NewTimer(time.Second * 3)          //nolint:gomnd // this is a test timeout
	ticker := time.NewTicker(time.Millisecond * 100) //nolint:gomnd // this is a test timeout

	defer timer.Stop()
	defer ticker.Stop()

	for {
		select {
		case <-timer.C:
			panic("timeout waiting for transfers to be removed from running list")
		default:
			ok1 := pipeline.ClientTransfers.Exists(s.ClientTrans.ID)
			ok2 := s.service.ManageTransfers().Exists(s.ClientTrans.ID + 1)

			if !ok1 && !ok2 {
				return
			}
		}
	}
}

func (s *SelfContext) AddTransferInfo(c convey.C, name string, val interface{}) {
	s.transferInfo[name] = val
	c.So(s.ClientTrans.SetTransferInfo(s.DB, s.transferInfo), convey.ShouldBeNil)
}

/*
func (s *SelfContext) AddFileInfo(c convey.C, name string, val interface{}) {
	c.So(s.ClientRule.IsSend, convey.ShouldBeTrue)
	s.fileInfo[name] = val
	c.So(s.ClientTrans.SetFileInfo(s.DB, s.fileInfo), convey.ShouldBeNil)
}
*/
