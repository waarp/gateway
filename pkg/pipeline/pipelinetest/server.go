package pipelinetest

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/proto"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

// ServerContext is a struct regrouping all the elements necessary for a server
// transfer test.
type ServerContext struct {
	*testData
	*serverData
	filename string
	constr   serviceConstructor
}

type serverData struct {
	Server     *model.LocalAgent
	LocAccount *model.LocalAccount
	ServerRule *model.Rule
}

func initServer(c convey.C, protocol string, constr serviceConstructor,
	servConf config.ProtoConfig,
) *ServerContext {
	t := initTestData(c)
	port := testhelpers.GetFreePort(c)
	server, locAcc := makeServerConf(c, t, port, protocol, servConf)

	return &ServerContext{
		testData: t,
		serverData: &serverData{
			Server:     server,
			LocAccount: locAcc,
		},
		filename: "test_server.file",
		constr:   constr,
	}
}

// Filename returns the expected name of the file used in server transfer tests.
func (s *ServerContext) Filename() string { return s.filename }

// InitServerPush creates a database and fills it with all the elements necessary
// for a server push transfer test of the given protocol. It then returns all these
// element inside a ServerContext.
func InitServerPush(c convey.C, protocol string, constr serviceConstructor,
	servConf config.ProtoConfig,
) *ServerContext {
	ctx := initServer(c, protocol, constr, servConf)
	ctx.ServerRule = makeServerPush(c, ctx.DB)

	return ctx
}

// InitServerPull creates a database and fills it with all the elements necessary
// for a server pull transfer test of the given protocol. It then returns all these
// element inside a ServerContext.
func InitServerPull(c convey.C, protocol string, constr serviceConstructor,
	servConf config.ProtoConfig,
) *ServerContext {
	ctx := initServer(c, protocol, constr, servConf)
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

func makeServerConf(c convey.C, data *testData, port uint16, protocol string,
	servConf config.ProtoConfig,
) (ag *model.LocalAgent, acc *model.LocalAccount) {
	jsonServConf := json.RawMessage(`{}`)

	if servConf != nil {
		var err error
		jsonServConf, err = json.Marshal(servConf)
		c.So(err, convey.ShouldBeNil)
	}

	rootDir := protocol + "_server_root"
	rootPath := mkURL(data.Paths.GatewayHome, rootDir)
	c.So(fs.MkdirAll(data.FS, rootPath), convey.ShouldBeNil)

	server := &model.LocalAgent{
		Name:          "server",
		Protocol:      protocol,
		RootDir:       rootDir,
		ProtoConfig:   jsonServConf,
		Address:       fmt.Sprintf("127.0.0.1:%d", port),
		ReceiveDir:    "receive",
		SendDir:       "send",
		TmpReceiveDir: "tmp",
	}

	c.So(data.DB.Insert(server).Run(), convey.ShouldBeNil)
	c.So(fs.MkdirAll(data.FS, rootPath.JoinPath(server.ReceiveDir)), convey.ShouldBeNil)
	c.So(fs.MkdirAll(data.FS, rootPath.JoinPath(server.SendDir)), convey.ShouldBeNil)
	c.So(fs.MkdirAll(data.FS, rootPath.JoinPath(server.TmpReceiveDir)), convey.ShouldBeNil)

	pswd := TestPassword
	if protocol == config.ProtocolR66 || protocol == config.ProtocolR66TLS {
		pswd = utils.R66Hash(pswd)
	}

	locAccount := &model.LocalAccount{
		LocalAgentID: server.ID,
		Login:        TestLogin,
		PasswordHash: hash(pswd),
	}

	c.So(data.DB.Insert(locAccount).Run(), convey.ShouldBeNil)

	return server, locAccount
}

// AddCryptos adds the given cryptos to the test database.
func (s *ServerContext) AddCryptos(c convey.C, certs ...model.Crypto) {
	for i := range certs {
		c.So(s.DB.Insert(&certs[i]).Run(), convey.ShouldBeNil)
	}
}

// StartService starts the service associated with the server defined in ServerContext.
func (s *ServerContext) StartService(c convey.C) proto.Service {
	logger := conf.GetLogger(fmt.Sprintf("test_%s_server", s.Server.Protocol))
	serv := s.constr(s.DB, logger)
	c.So(serv.Start(s.Server), convey.ShouldBeNil)
	c.Reset(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		c.So(serv.Stop(ctx), convey.ShouldBeNil)
	})

	return serv
}

// CheckTransferOK checks if the client transfer history entry has succeeded as
// expected.
func (s *ServerContext) CheckTransferOK(c convey.C) {
	var actual model.HistoryEntry

	c.So(s.DB.Get(&actual, "id=?", 1).Run(), convey.ShouldBeNil)

	remoteID := actual.RemoteTransferID
	progress := TestFileSize
	s.checkServerTransferOK(c, remoteID, s.filename, progress, s.testData, &actual, nil)
}
