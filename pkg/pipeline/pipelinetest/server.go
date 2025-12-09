package pipelinetest

import (
	"context"
	"encoding/json"
	"path"
	"time"

	"github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

// ServerContext is a struct regrouping all the elements necessary for a server
// transfer test.
type ServerContext struct {
	*testData
	*serverData

	filename string
	service  testService
}

type serverData struct {
	Server     *model.LocalAgent
	LocAccount *model.LocalAccount
	ServerRule *model.Rule
}

type testService interface {
	protocol.Server
	SetTracer(getTrace func() pipeline.Trace)
}

func initServer(c convey.C, proto string, servConf protocol.ServerConfig,
) *ServerContext {
	t := initTestData(c)
	port := testhelpers.GetFreePort(c)
	locAg, locAcc := makeServerConf(c, t, port, proto, servConf)

	constr := Protocols[proto].MakeServer
	server := constr(t.DB, locAg)

	testServer, ok := server.(testService)
	c.So(ok, convey.ShouldBeTrue)

	return &ServerContext{
		testData: t,
		serverData: &serverData{
			Server:     locAg,
			LocAccount: locAcc,
		},
		filename: "test_server.file",
		service:  testServer,
	}
}

// Filename returns the expected name of the file used in server transfer tests.
func (s *ServerContext) Filename() string { return s.filename }

// InitServerPush creates a database and fills it with all the elements necessary
// for a server push transfer test of the given protocol. It then returns all these
// element inside a ServerContext.
func InitServerPush(c convey.C, proto string, servConf protocol.ServerConfig,
) *ServerContext {
	ctx := initServer(c, proto, servConf)
	ctx.ServerRule = makeServerPush(c, ctx.DB)

	return ctx
}

// InitServerPull creates a database and fills it with all the elements necessary
// for a server pull transfer test of the given protocol. It then returns all these
// element inside a ServerContext.
func InitServerPull(c convey.C, proto string, servConf protocol.ServerConfig,
) *ServerContext {
	ctx := initServer(c, proto, servConf)
	ctx.ServerRule = makeServerPull(c, ctx.DB)

	return ctx
}

func makeServerPush(c convey.C, db *database.DB) *model.Rule {
	rule := &model.Rule{
		Name:           "PUSH",
		IsSend:         false,
		Path:           "push",
		LocalDir:       "serv_push_dir",
		TmpLocalRcvDir: "serv_push_tmp",
	}
	c.So(db.Insert(rule).Run(), convey.ShouldBeNil)
	makeRuleTasks(c, db, rule)

	return rule
}

func makeServerPull(c convey.C, db *database.DB) *model.Rule {
	rule := &model.Rule{
		Name:     "PULL",
		IsSend:   true,
		Path:     "pull",
		LocalDir: "serv_pull_dir",
	}
	c.So(db.Insert(rule).Run(), convey.ShouldBeNil)
	makeRuleTasks(c, db, rule)

	return rule
}

func makeServerConf(c convey.C, data *testData, port uint16, proto string,
	servConf protocol.ServerConfig,
) (ag *model.LocalAgent, acc *model.LocalAccount) {
	jsonServConf := map[string]any{}

	if servConf != nil {
		err := utils.JSONConvert(servConf, &jsonServConf)
		c.So(err, convey.ShouldBeNil)
	}

	rootDir := proto + "_server_root"
	rootPath := path.Join(data.Paths.GatewayHome, rootDir)
	c.So(fs.MkdirAll(rootPath), convey.ShouldBeNil)

	server := &model.LocalAgent{
		Name:          proto + "-server",
		Protocol:      proto,
		RootDir:       rootDir,
		ProtoConfig:   jsonServConf,
		Address:       types.Addr("127.0.0.1", port),
		ReceiveDir:    "receive",
		SendDir:       "send",
		TmpReceiveDir: "tmp",
	}

	c.So(data.DB.Insert(server).Run(), convey.ShouldBeNil)
	c.So(fs.MkdirAll(path.Join(rootPath, server.ReceiveDir)), convey.ShouldBeNil)
	c.So(fs.MkdirAll(path.Join(rootPath, server.SendDir)), convey.ShouldBeNil)
	c.So(fs.MkdirAll(path.Join(rootPath, server.TmpReceiveDir)), convey.ShouldBeNil)

	locAccount := &model.LocalAccount{
		LocalAgentID: server.ID,
		Login:        TestLogin,
	}
	c.So(data.DB.Insert(locAccount).Run(), convey.ShouldBeNil)

	cred := &model.Credential{
		LocalAccountID: utils.NewNullInt64(locAccount.ID),
		Type:           auth.Password,
		Value:          TestPassword,
	}
	c.So(data.DB.Insert(cred).Run(), convey.ShouldBeNil)

	return server, locAccount
}

// AddAuths adds the given cryptos to the test database.
func (s *ServerContext) AddAuths(c convey.C, creds ...*model.Credential) {
	for _, cred := range creds {
		c.So(s.DB.Insert(cred).Run(), convey.ShouldBeNil)
	}
}

// StartService starts the service associated with the server defined in ServerContext.
func (s *ServerContext) StartService(c convey.C) {
	c.So(s.service.Start(), convey.ShouldBeNil)
	c.Reset(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		c.So(s.service.Stop(ctx), convey.ShouldBeNil)
	})

	s.service.SetTracer(s.makeServerTracer(s.ServerRule.IsSend))
}

// CheckTransferOK checks if the client transfer history entry has succeeded as
// expected.
func (s *ServerContext) CheckTransferOK(c convey.C) {
	var actual model.HistoryEntry

	c.So(s.DB.Get(&actual, "id=?", 1).Run(), convey.ShouldBeNil)

	remoteID := actual.RemoteTransferID
	progress := TestFileSize
	s.checkServerTransferOK(c, remoteID, s.filename, progress, s.testData, &actual,
		map[string]any{model.FollowID: json.Number(remoteID)})
}
