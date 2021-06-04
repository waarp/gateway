package pipelinetest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"
	"github.com/smartystreets/goconvey/convey"
)

// ServerContext is a struct regrouping all the elements necessary for a server
// transfer test.
type ServerContext struct {
	*testData
	*serverData
}

type serverData struct {
	Server     *model.LocalAgent
	LocAccount *model.LocalAccount
	Rule       *model.Rule
}

func initServer(c convey.C, proto string, servConf config.ProtoConfig) *ServerContext {
	t := initTestData(c)
	port := testhelpers.GetFreePort(c)
	server, locAcc := makeServerConf(c, t.DB, port, t.Paths.GatewayHome, proto, servConf)

	return &ServerContext{
		testData: t,
		serverData: &serverData{
			Server:     server,
			LocAccount: locAcc,
		},
	}
}

// InitServerPush creates a database and fills it with all the elements necessary
// for a server push transfer test of the given protocol. It then returns all these
// element inside a ServerContext.
func InitServerPush(c convey.C, proto string, servConf config.ProtoConfig) *ServerContext {
	ctx := initServer(c, proto, servConf)
	ctx.Rule = makeServerPush(c, ctx.DB)
	return ctx
}

// InitServerPull creates a database and fills it with all the elements necessary
// for a server pull transfer test of the given protocol. It then returns all these
// element inside a ServerContext.
func InitServerPull(c convey.C, proto string, servConf config.ProtoConfig) *ServerContext {
	ctx := initServer(c, proto, servConf)
	ctx.Rule = makeServerPull(c, ctx.DB)
	return ctx
}

func makeServerPush(c convey.C, db *database.DB) *model.Rule {
	rule := &model.Rule{
		Name:   "PUSH",
		IsSend: false,
		Path:   "/push",
	}
	c.So(db.Insert(rule).Run(), convey.ShouldBeNil)
	makeRuleTasks(c, db, rule, false)
	return rule
}

func makeServerPull(c convey.C, db *database.DB) *model.Rule {
	rule := &model.Rule{
		Name:   "PULL",
		IsSend: true,
		Path:   "/pull",
	}
	c.So(db.Insert(rule).Run(), convey.ShouldBeNil)
	makeRuleTasks(c, db, rule, false)
	return rule
}

func makeServerConf(c convey.C, db *database.DB, port uint16, home, proto string,
	servConf config.ProtoConfig) (ag *model.LocalAgent, acc *model.LocalAccount) {

	jsonServConf := json.RawMessage(`{}`)
	if servConf != nil {
		var err error
		jsonServConf, err = json.Marshal(servConf)
		c.So(err, convey.ShouldBeNil)
	}

	root := filepath.Join(home, proto+"_server_root")
	c.So(os.MkdirAll(root, 0o700), convey.ShouldBeNil)

	server := &model.LocalAgent{
		Name:        "server",
		Protocol:    proto,
		Root:        utils.ToStandardPath(root),
		ProtoConfig: jsonServConf,
		Address:     fmt.Sprintf("localhost:%d", port),
	}
	c.So(db.Insert(server).Run(), convey.ShouldBeNil)
	c.So(os.MkdirAll(filepath.Join(root, server.LocalInDir), 0o700), convey.ShouldBeNil)
	c.So(os.MkdirAll(filepath.Join(root, server.LocalOutDir), 0o700), convey.ShouldBeNil)
	c.So(os.MkdirAll(filepath.Join(root, server.LocalTmpDir), 0o700), convey.ShouldBeNil)

	locAccount := &model.LocalAccount{
		LocalAgentID: server.ID,
		Login:        TestLogin,
		PasswordHash: hash(TestPassword),
	}
	c.So(db.Insert(locAccount).Run(), convey.ShouldBeNil)

	return server, locAccount
}

func (s *ServerContext) AddCerts(c convey.C, certs ...model.Crypto) {
	for _, cert := range certs {
		c.So(s.DB.Insert(&cert).Run(), convey.ShouldBeNil)
	}
}
