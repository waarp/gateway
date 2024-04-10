// Package pipelinetest regroups a series of utility functions and structs for
// quickly instantiating & running transfer pipelines for test purposes.
package pipelinetest

import (
	"context"
	"path"
	"strings"
	"time"

	"github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/controller"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks/taskstest"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

// SelfContext is a struct regrouping all the necessary elements to perform
// self-transfer tests, including transfer failure tests.
type SelfContext struct {
	*testData
	*clientData
	*serverData
	*transData

	ServerService testService
	ClientService protocol.Client
	fail          *model.Task
	protoFeatures *ProtoFeatures
}

func initSelfTransfer(c convey.C, proto string, clientConf protocol.ClientConfig,
	partConf protocol.PartnerConfig, servConf protocol.ServerConfig,
) *SelfContext {
	feat, protoExists := Protocols[proto]
	c.SoMsg("the protocol should exist", protoExists, convey.ShouldBeTrue)

	test := initTestData(c)
	port := testhelpers.GetFreePort(c)
	cli, remAg, remAcc := makeClientConf(c, test.DB, port, proto, clientConf, partConf)
	locAg, locAcc := makeServerConf(c, test, port, proto, servConf)

	client := feat.MakeClient(test.DB, cli)
	c.So(client.Start(), convey.ShouldBeNil)
	services.Clients[cli.Name] = client

	c.Reset(func() {
		delete(services.Clients, cli.Name)

		//nolint:gomnd //this is just for tests
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := client.Stop(ctx); err != nil {
			testhelpers.TestLogger(c, cli.Name).Warning(
				"Error while stopping client: %v", err)
		}
	})

	server := feat.MakeServer(test.DB, locAg)

	testServer, ok := server.(testService)
	c.So(ok, convey.ShouldBeTrue)

	return &SelfContext{
		testData: test,
		clientData: &clientData{
			Partner:     remAg,
			RemAccount:  remAcc,
			Client:      cli,
			ProtoClient: client,
		},
		serverData: &serverData{
			Server:     locAg,
			LocAccount: locAcc,
		},
		transData: &transData{
			transferInfo: map[string]interface{}{},
			// fileInfo:     map[string]interface{}{},
		},
		ServerService: testServer,
		ClientService: client,
		protoFeatures: &feat,
	}
}

// InitSelfPushTransfer creates a database and fills it with all the elements
// necessary for a push self-transfer test of the given protocol. It then returns
// all these elements inside a SelfContext.
func InitSelfPushTransfer(c convey.C, proto string, clientConf protocol.ClientConfig,
	partConf protocol.PartnerConfig, servConf protocol.ServerConfig,
) *SelfContext {
	ctx := initSelfTransfer(c, proto, clientConf, partConf, servConf)
	ctx.ClientRule = makeClientPush(c, ctx.DB, proto)
	ctx.ServerRule = makeServerPush(c, ctx.DB)
	ctx.addPushTransfer(c)

	return ctx
}

// InitSelfPullTransfer creates a database and fills it with all the elements
// necessary for a pull self-transfer test of the given protocol. It then returns
// all these elements inside a SelfContext.
func InitSelfPullTransfer(c convey.C, proto string, clientConf protocol.ClientConfig,
	partConf protocol.PartnerConfig, servConf protocol.ServerConfig,
) *SelfContext {
	ctx := initSelfTransfer(c, proto, clientConf, partConf, servConf)
	ctx.ClientRule = makeClientPull(c, ctx.DB, proto)
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
		ClientID:        utils.NewNullInt64(s.Client.ID),
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
		ClientID:        utils.NewNullInt64(s.Client.ID),
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
	const shutdownTimeout = 5 * time.Second

	c.So(s.ServerService.Start(), convey.ShouldBeNil)
	c.Reset(func() {
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		c.So(s.ServerService.Stop(ctx), convey.ShouldBeNil)
	})
}

// AddCreds adds the given credentials to the test database.
func (s *SelfContext) AddCreds(c convey.C, creds ...*model.Credential) {
	for _, cred := range creds {
		c.So(s.DB.Insert(cred).Run(), convey.ShouldBeNil)
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
const DataErrorOffset = TestFileSize / 4

func (s *SelfContext) AddClientDataError(_ convey.C) {
	s.hasClientDataError = true
}

func (s *SelfContext) AddServerDataError(_ convey.C) {
	s.hasServerDataError = true
}

// RunTransfer executes the test self-transfer in its entirety.
func (s *SelfContext) RunTransfer(c convey.C, willFail bool) {
	pip, err := controller.NewClientPipeline(s.DB, s.ClientTrans)
	c.So(err, convey.ShouldBeNil)
	s.setTrace(pip.Pip)

	if tErr := pip.Run(); !willFail {
		convey.So(tErr, convey.ShouldBeNil)
	}

	const transferTimeout = 10 * time.Second

	c.SoMsg("The client pipeline should have ended",
		utils.WaitChan(s.cliDone, transferTimeout), convey.ShouldBeTrue)
	c.SoMsg("The server pipeline should have ended",
		utils.WaitChan(s.servDone, transferTimeout), convey.ShouldBeTrue)
	s.waitForListDeletion()
}

func (s *SelfContext) setTrace(pip *pipeline.Pipeline) {
	s.setClientTrace(pip)
	s.ServerService.SetTracer(s.makeServerTracer(s.ServerRule.IsSend))
}

func (s *SelfContext) resetTransfer(c convey.C) {
	s.hasServerDataError = false
	s.hasClientDataError = false
	s.cliDone = make(chan bool)
	s.servDone = make(chan bool)

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

	if !s.protoFeatures.RuleName {
		ruleDir = strings.TrimPrefix(ruleDir, s.ServerRule.Path+"/")
	}

	return ruleDir
}

func (s *SelfContext) checkServerTransferOK(c convey.C, actual *model.HistoryEntry) {
	remoteID := s.transData.ClientTrans.RemoteTransferID
	if !s.protoFeatures.TransID {
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

	c.SoMsg("Then the database ID should match",
		actual.ID, convey.ShouldEqual, s.ClientTrans.ID)
	c.SoMsg("Then the remote transfer ID should not be empty",
		actual.RemoteTransferID, convey.ShouldNotBeBlank)
	c.SoMsg("Then the transfer should be owned by the gateway",
		actual.Owner, convey.ShouldEqual, conf.GlobalConfig.GatewayName)
	c.SoMsg("Then the transfer should be in error",
		actual.Status, convey.ShouldEqual, types.StatusError)
	c.SoMsg("Then the rule ID should match",
		actual.RuleID, convey.ShouldEqual, s.ClientRule.ID)
	c.SoMsg("Then the client ID should match",
		actual.ClientID.Int64, convey.ShouldEqual, s.Client.ID)
	c.SoMsg("Then the remote account ID should match",
		actual.RemoteAccountID.Int64, convey.ShouldEqual, s.RemAccount.ID)

	c.SoMsg("Then the error code should match",
		actual.ErrCode, convey.ShouldResemble, errCode)
	c.SoMsg("Then the error details should match",
		actual.ErrDetails, convey.ShouldResemble, errMsg)
	c.SoMsg("Then the file size should either match or be unknown",
		actual.Filesize, testhelpers.ShouldBeOneOf, model.UnknownSize, TestFileSize)
	c.SoMsg("Then the transfer should have a reasonable progression",
		actual.Progress, convey.ShouldBeBetweenOrEqual, 0, TestFileSize)
	c.SoMsg("Then the transfer should be in one of the expected steps",
		actual.Step.String(), testhelpers.ShouldBeOneOf, stepsStr)

	if actual.Step == types.StepPreTasks || actual.Step == types.StepPostTasks {
		c.SoMsg("Then the task counter should be 1",
			actual.TaskNumber, convey.ShouldEqual, 1)
	} else {
		c.SoMsg("Then the task counter should be 0",
			actual.TaskNumber, convey.ShouldEqual, 0)
	}
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

	c.SoMsg("Then the database ID should match",
		actual.ID, convey.ShouldEqual, id)
	c.SoMsg("Then the transfer should be owned by the gateway",
		actual.Owner, convey.ShouldEqual, conf.GlobalConfig.GatewayName)
	c.SoMsg("Then the transfer should be in error",
		actual.Status, convey.ShouldEqual, types.StatusError)
	c.SoMsg("Then the rule ID should match",
		actual.RuleID, convey.ShouldEqual, s.ServerRule.ID)
	c.SoMsg("Then the local account ID should match",
		actual.LocalAccountID.Int64, convey.ShouldEqual, s.LocAccount.ID)

	c.SoMsg("Then the error code should match",
		actual.ErrCode, convey.ShouldResemble, errCode)
	c.SoMsg("Then the error details should match",
		actual.ErrDetails, convey.ShouldResemble, errMsg)
	c.SoMsg("Then the file size should either match or be unknown",
		actual.Filesize, testhelpers.ShouldBeOneOf, model.UnknownSize, TestFileSize)
	c.SoMsg("Then the transfer should have a reasonable progression",
		actual.Progress, convey.ShouldBeBetweenOrEqual, 0, TestFileSize)
	c.SoMsg("Then the transfer should be in one of the expected steps",
		actual.Step.String(), testhelpers.ShouldBeOneOf, stepsStr)

	if s.protoFeatures.TransID {
		c.SoMsg("Then the remote transfer ID should match",
			actual.RemoteTransferID, convey.ShouldEqual, s.ClientTrans.RemoteTransferID)
	}

	if actual.Step == types.StepPreTasks || actual.Step == types.StepPostTasks {
		c.SoMsg("Then the task counter should be 1",
			actual.TaskNumber, convey.ShouldEqual, 1)
	} else {
		c.SoMsg("Then the task counter should be 0",
			actual.TaskNumber, convey.ShouldEqual, 0)
	}
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
			ok1 := pipeline.List.Exists(s.ClientTrans.ID)
			ok2 := pipeline.List.Exists(s.ClientTrans.ID + 1)

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

func (s *SelfContext) GetTransferContext(c convey.C) *model.TransferContext {
	partnerCryptos, err := s.Partner.GetCredentials(s.DB)
	c.So(err, convey.ShouldBeNil)

	accountCryptos, err := s.RemAccount.GetCredentials(s.DB)
	c.So(err, convey.ShouldBeNil)

	return &model.TransferContext{
		Transfer:           s.transData.ClientTrans,
		TransInfo:          s.transData.transferInfo,
		Rule:               s.ClientRule,
		Client:             s.Client,
		RemoteAgent:        s.Partner,
		RemoteAccount:      s.RemAccount,
		RemoteAgentCreds:   partnerCryptos,
		RemoteAccountCreds: accountCryptos,
		Paths:              s.Paths,
	}
}
