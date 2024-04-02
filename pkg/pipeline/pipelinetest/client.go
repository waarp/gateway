package pipelinetest

import (
	"fmt"
	"path"
	"sync/atomic"
	"time"

	"github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/controller"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

type clientData struct {
	Partner    *model.RemoteAgent
	RemAccount *model.RemoteAccount
	Client     *model.Client
	ClientRule *model.Rule

	ProtoClient protocol.Client
}

// ClientContext is a struct regrouping all the elements necessary for a
// client transfer test.
type ClientContext struct {
	*testData
	*clientData
	*transData
	protoFeatures *ProtoFeatures
}

func initClient(c convey.C, proto string, clientConf protocol.ClientConfig,
	partConf protocol.PartnerConfig,
) *ClientContext {
	feat, ok := Protocols[proto]
	c.So(ok, convey.ShouldBeTrue)
	t := initTestData(c)
	port := testhelpers.GetFreePort(c)
	cli, partner, remAcc := makeClientConf(c, t.DB, port, proto, clientConf, partConf)

	constr := Protocols[proto].MakeClient

	client := constr(t.DB, cli)
	c.So(client.Start(), convey.ShouldBeNil)

	services.Clients[cli.Name] = client

	return &ClientContext{
		testData: t,
		clientData: &clientData{
			Partner:     partner,
			RemAccount:  remAcc,
			Client:      cli,
			ProtoClient: client,
		},
		transData: &transData{
			transferInfo: map[string]interface{}{},
			// fileInfo:     map[string]interface{}{},
		},
		protoFeatures: &feat,
	}
}

// InitClientPush creates a database and fills it with all the elements necessary
// for a push client transfer test of the given protocol. It then returns all these
// elements inside a ClientContext.
func InitClientPush(c convey.C, proto string, clientConf protocol.ClientConfig,
	partConf protocol.PartnerConfig,
) *ClientContext {
	ctx := initClient(c, proto, clientConf, partConf)
	ctx.ClientRule = makeClientPush(c, ctx.DB, proto)
	ctx.addPushTransfer(c)

	return ctx
}

// InitClientPull creates a database and fills it with all the elements necessary
// for a pull client transfer test of the given protocol. It then returns all these
// element inside a ClientContext.
func InitClientPull(c convey.C, proto string, cont []byte,
	clientConf protocol.ClientConfig, partConf protocol.PartnerConfig,
) *ClientContext {
	ctx := initClient(c, proto, clientConf, partConf)
	ctx.ClientRule = makeClientPull(c, ctx.DB, proto)
	ctx.addPullTransfer(c, cont)

	return ctx
}

// AddCryptos adds the given cryptos to the test database.
func (cc *ClientContext) AddCryptos(c convey.C, certs ...model.Crypto) {
	for i := range certs {
		c.So(cc.DB.Insert(&certs[i]).Run(), convey.ShouldBeNil)
	}
}

func makeClientPush(c convey.C, db *database.DB, proto string) *model.Rule {
	rule := &model.Rule{
		Name:      "PUSH",
		IsSend:    true,
		LocalDir:  "cli_push_loc",
		RemoteDir: "cli_push_rem",
	}

	if !Protocols[proto].RuleName {
		rule.RemoteDir = path.Join("push", rule.RemoteDir)
	}

	c.So(db.Insert(rule).Run(), convey.ShouldBeNil)
	makeRuleTasks(c, db, rule)

	return rule
}

func makeClientPull(c convey.C, db *database.DB, proto string) *model.Rule {
	rule := &model.Rule{
		Name:           "PULL",
		IsSend:         false,
		LocalDir:       "cli_pull_loc",
		TmpLocalRcvDir: "cli_pull_tmp",
		RemoteDir:      "cli_pull_rem",
	}

	if !Protocols[proto].RuleName {
		rule.RemoteDir = path.Join("pull", rule.RemoteDir)
	}

	c.So(db.Insert(rule).Run(), convey.ShouldBeNil)
	makeRuleTasks(c, db, rule)

	return rule
}

func makeClientConf(c convey.C, db *database.DB, port uint16, proto string,
	clientConf protocol.ClientConfig, partConf protocol.PartnerConfig,
) (*model.Client, *model.RemoteAgent, *model.RemoteAccount) {
	jsonClientConf := map[string]any{}
	jsonPartConf := map[string]any{}

	if clientConf != nil {
		err := utils.JSONConvert(clientConf, &jsonClientConf)
		c.So(err, convey.ShouldBeNil)
	}

	if partConf != nil {
		err := utils.JSONConvert(partConf, &jsonPartConf)
		c.So(err, convey.ShouldBeNil)
	}

	client := &model.Client{
		Name:        "client",
		Protocol:    proto,
		ProtoConfig: jsonClientConf,
	}
	c.So(db.Insert(client).Run(), convey.ShouldBeNil)

	partner := &model.RemoteAgent{
		Name:        "partner",
		Protocol:    proto,
		ProtoConfig: jsonPartConf,
		Address:     fmt.Sprintf("127.0.0.1:%d", port),
	}
	c.So(db.Insert(partner).Run(), convey.ShouldBeNil)

	remAccount := &model.RemoteAccount{
		RemoteAgentID: partner.ID,
		Login:         TestLogin,
		Password:      TestPassword,
	}
	c.So(db.Insert(remAccount).Run(), convey.ShouldBeNil)

	return client, partner, remAccount
}

//nolint:dupl // factorizing would hurt readability
func (cc *ClientContext) addPushTransfer(c convey.C) {
	testFile := mkURL(cc.Paths.GatewayHome, cc.ClientRule.LocalDir, "self_transfer_push")
	cc.fileContent = AddSourceFile(c, cc.FS, testFile)

	trans := &model.Transfer{
		RuleID:          cc.ClientRule.ID,
		ClientID:        utils.NewNullInt64(cc.Client.ID),
		RemoteAccountID: utils.NewNullInt64(cc.RemAccount.ID),
		SrcFilename:     "self_transfer_push",
		Start:           time.Now(),
	}
	c.So(cc.DB.Insert(trans).Run(), convey.ShouldBeNil)

	cc.ClientTrans = trans
}

//nolint:dupl // factorizing would hurt readability
func (cc *ClientContext) addPullTransfer(c convey.C, cont []byte) {
	cc.fileContent = cont

	trans := &model.Transfer{
		RuleID:          cc.ClientRule.ID,
		ClientID:        utils.NewNullInt64(cc.Client.ID),
		RemoteAccountID: utils.NewNullInt64(cc.RemAccount.ID),
		SrcFilename:     "self_transfer_pull",
		Filesize:        model.UnknownSize,
		Start:           time.Now(),
	}
	c.So(cc.DB.Insert(trans).Run(), convey.ShouldBeNil)

	cc.ClientTrans = trans
}

// RunTransfer executes the test self-transfer in its entirety.
func (cc *ClientContext) RunTransfer(c convey.C) {
	pip, err := controller.NewClientPipeline(cc.DB, cc.ClientTrans)
	c.So(err, convey.ShouldBeNil)
	cc.setClientTrace(pip.Pip)

	convey.So(pip.Run(), convey.ShouldBeNil)

	ok := pipeline.List.Exists(cc.ClientTrans.ID)
	c.So(ok, convey.ShouldBeFalse)
}

// CheckTransferOK checks if the client transfer history entry has succeeded as
// expected.
func (cc *ClientContext) CheckTransferOK(c convey.C) {
	var actual model.HistoryEntry

	c.So(cc.DB.Get(&actual, "id=?", cc.ClientTrans.ID).Run(), convey.ShouldBeNil)
	cc.checkClientTransferOK(c, cc.transData, cc.DB, &actual)
}

func (cc *ClientContext) GetTransferContent(c convey.C) *model.TransferContext {
	partnerCryptos, err := cc.Partner.GetCryptos(cc.DB)
	c.So(err, convey.ShouldBeNil)

	accountCryptos, err := cc.RemAccount.GetCryptos(cc.DB)
	c.So(err, convey.ShouldBeNil)

	return &model.TransferContext{
		Transfer:             cc.ClientTrans,
		TransInfo:            cc.transferInfo,
		Rule:                 cc.ClientRule,
		Client:               cc.Client,
		RemoteAgent:          cc.Partner,
		RemoteAgentCryptos:   partnerCryptos,
		RemoteAccount:        cc.RemAccount,
		RemoteAccountCryptos: accountCryptos,
		Paths:                cc.Paths,
	}
}

func (cc *ClientContext) setClientTrace(pip *pipeline.Pipeline) {
	pip.Trace.OnPreTask = func(int8) error {
		atomic.AddUint32(&cc.cliPreTasksNb, 1)

		return nil
	}

	pip.Trace.OnPostTask = func(int8) error {
		atomic.AddUint32(&cc.cliPostTasksNb, 1)

		return nil
	}

	pip.Trace.OnErrorTask = func(int8) {
		atomic.AddUint32(&cc.cliErrTasksNb, 1)
	}

	pip.Trace.OnTransferEnd = func() { close(cc.cliDone) }
}
