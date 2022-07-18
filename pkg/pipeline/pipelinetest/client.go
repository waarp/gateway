package pipelinetest

import (
	"encoding/json"
	"fmt"
	"path"
	"sync/atomic"
	"time"

	"github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

type clientData struct {
	Partner    *model.RemoteAgent
	RemAccount *model.RemoteAccount
	ClientRule *model.Rule
}

// ClientContext is a struct regrouping all the elements necessary for a
// client transfer test.
type ClientContext struct {
	*testData
	*clientData
	*transData
	protoFeatures *features
}

func initClient(c convey.C, proto string, partConf config.PartnerProtoConfig) *ClientContext {
	feat, ok := protocols[proto]
	c.So(ok, convey.ShouldBeTrue)
	t := initTestData(c)
	port := testhelpers.GetFreePort(c)
	partner, remAcc := makeClientConf(c, t.DB, port, proto, partConf)

	return &ClientContext{
		testData: t,
		clientData: &clientData{
			Partner:    partner,
			RemAccount: remAcc,
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
func InitClientPush(c convey.C, proto string, partConf config.PartnerProtoConfig,
) *ClientContext {
	ctx := initClient(c, proto, partConf)
	ctx.ClientRule = makeClientPush(c, ctx.DB, proto)
	ctx.addPushTransfer(c)

	return ctx
}

// InitClientPull creates a database and fills it with all the elements necessary
// for a pull client transfer test of the given protocol. It then returns all these
// element inside a ClientContext.
func InitClientPull(c convey.C, proto string, cont []byte, partConf config.PartnerProtoConfig,
) *ClientContext {
	ctx := initClient(c, proto, partConf)
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

	if !protocols[proto].ruleName {
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

	if !protocols[proto].ruleName {
		rule.RemoteDir = path.Join("pull", rule.RemoteDir)
	}

	c.So(db.Insert(rule).Run(), convey.ShouldBeNil)
	makeRuleTasks(c, db, rule)

	return rule
}

func makeClientConf(c convey.C, db *database.DB, port uint16, proto string,
	partConf config.PartnerProtoConfig,
) (*model.RemoteAgent, *model.RemoteAccount) {
	jsonPartConf := json.RawMessage(`{}`)

	if partConf != nil {
		var err error
		jsonPartConf, err = json.Marshal(partConf)
		c.So(err, convey.ShouldBeNil)
	}

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

	return partner, remAccount
}

//nolint:dupl // factorizing would hurt readability
func (cc *ClientContext) addPushTransfer(c convey.C) {
	testFile := mkURL(cc.Paths.GatewayHome, cc.ClientRule.LocalDir, "self_transfer_push")
	cc.fileContent = AddSourceFile(c, cc.FS, testFile)

	trans := &model.Transfer{
		RuleID:          cc.ClientRule.ID,
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
	pip, err := pipeline.NewClientPipeline(cc.DB, cc.ClientTrans)
	c.So(err, convey.ShouldBeNil)
	cc.setClientTrace(pip.Pip)

	convey.So(pip.Run(), convey.ShouldBeNil)

	ok := pipeline.ClientTransfers.Exists(cc.ClientTrans.ID)
	c.So(ok, convey.ShouldBeFalse)
}

// CheckTransferOK checks if the client transfer history entry has succeeded as
// expected.
func (cc *ClientContext) CheckTransferOK(c convey.C) {
	var actual model.HistoryEntry

	c.So(cc.DB.Get(&actual, "id=?", cc.ClientTrans.ID).Run(), convey.ShouldBeNil)
	cc.checkClientTransferOK(c, cc.transData, cc.DB, &actual)
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
