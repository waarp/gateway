package pipelinetest

import (
	"encoding/json"
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/smartystreets/goconvey/convey"
)

type clientData struct {
	Partner    *model.RemoteAgent
	RemAccount *model.RemoteAccount
}

// ClientContext is a struct regrouping all the elements necessary for a
// client transfer test.
type ClientContext struct {
	*testData
	*clientData
	Rule *model.Rule
	*transData
}

func initClient(c convey.C, proto string, partConf config.ProtoConfig) *ClientContext {
	t := initTestData(c)
	port := testhelpers.GetFreePort(c)
	partner, remAcc := makeClientConf(c, t.DB, port, proto, partConf)

	return &ClientContext{
		testData: t,
		clientData: &clientData{
			Partner:    partner,
			RemAccount: remAcc,
		},
	}
}

// InitClientPush creates a database and fills it with all the elements necessary
// for a push client transfer test of the given protocol. It then returns all these
// element inside a ClientContext.
func InitClientPush(c convey.C, proto string, partConf config.ProtoConfig) *ClientContext {
	ctx := initClient(c, proto, partConf)
	ctx.Rule = makeClientPush(c, ctx.DB)
	return ctx
}

// InitClientPull creates a database and fills it with all the elements necessary
// for a pull client transfer test of the given protocol. It then returns all these
// element inside a ClientContext.
func InitClientPull(c convey.C, proto string, partConf config.ProtoConfig) *ClientContext {
	ctx := initClient(c, proto, partConf)
	ctx.Rule = makeClientPull(c, ctx.DB)
	return ctx
}

func makeClientPush(c convey.C, db *database.DB) *model.Rule {
	rule := &model.Rule{
		Name:      "PUSH",
		IsSend:    true,
		Path:      "/push",
		RemoteDir: "/push",
	}
	c.So(db.Insert(rule).Run(), convey.ShouldBeNil)
	makeRuleTasks(c, db, rule)
	return rule
}

func makeClientPull(c convey.C, db *database.DB) *model.Rule {
	rule := &model.Rule{
		Name:      "PULL",
		IsSend:    false,
		Path:      "/pull",
		RemoteDir: "/pull",
	}
	c.So(db.Insert(rule).Run(), convey.ShouldBeNil)
	makeRuleTasks(c, db, rule)
	return rule
}

func makeClientConf(c convey.C, db *database.DB, port uint16, proto string,
	partConf config.ProtoConfig) (*model.RemoteAgent, *model.RemoteAccount) {

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
		Address:     fmt.Sprintf("localhost:%d", port),
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
