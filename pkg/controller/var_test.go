package controller

import (
	"path"

	"code.waarp.fr/lib/log"
	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocolstest"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

const testProtocol = "test_proto"

//nolint:gochecknoinits // init is used to ease the tests
func init() {
	protocols.Register(testProtocol, protocolstest.TestModule{})
}

type testContext struct {
	root   string
	db     *database.DB
	logger *log.Logger

	client        *model.Client
	partner       *model.RemoteAgent
	remoteAccount *model.RemoteAccount
	server        *model.LocalAgent
	localAccount  *model.LocalAccount

	send *model.Rule
	recv *model.Rule
}

func initTestDB(c C, rootPath string) *testContext {
	db := database.TestDatabase(c)
	c.So(logging.AddLogBackend("DEBUG", "stdout", "", ""), ShouldBeNil)
	logger := testhelpers.TestLogger(c, "Pipeline test")

	paths := conf.PathsConfig{
		GatewayHome:   rootPath,
		DefaultInDir:  "in",
		DefaultOutDir: "out",
		DefaultTmpDir: "work",
	}
	conf.GlobalConfig.Paths = paths

	So(fs.MkdirAll(path.Join(rootPath, paths.DefaultInDir)), ShouldBeNil)
	So(fs.MkdirAll(path.Join(rootPath, paths.DefaultOutDir)), ShouldBeNil)
	So(fs.MkdirAll(path.Join(rootPath, paths.DefaultTmpDir)), ShouldBeNil)

	send := &model.Rule{
		Name:      "send",
		IsSend:    true,
		Path:      "/send",
		LocalDir:  "sLocal",
		RemoteDir: "sRemote",
	}
	recv := &model.Rule{
		Name:           "recv",
		IsSend:         false,
		Path:           "/recv",
		LocalDir:       "rLocal",
		RemoteDir:      "rRemote",
		TmpLocalRcvDir: "rTmp",
	}
	c.So(db.Insert(recv).Run(), ShouldBeNil)
	c.So(db.Insert(send).Run(), ShouldBeNil)

	So(fs.MkdirAll(path.Join(rootPath, send.LocalDir)), ShouldBeNil)
	So(fs.MkdirAll(path.Join(rootPath, recv.LocalDir)), ShouldBeNil)
	So(fs.MkdirAll(path.Join(rootPath, recv.TmpLocalRcvDir)), ShouldBeNil)

	server := &model.LocalAgent{
		Name: "server", Protocol: testProtocol,
		Address: types.Addr("localhost", 1111),
	}
	So(db.Insert(server).Run(), ShouldBeNil)

	locAccount := &model.LocalAccount{
		LocalAgentID: server.ID,
		Login:        "toto",
	}
	So(db.Insert(locAccount).Run(), ShouldBeNil)

	client := &model.Client{
		Name: "client", Protocol: testProtocol,
		LocalAddress: types.Addr("127.0.0.1", 2000),
	}
	So(db.Insert(client).Run(), ShouldBeNil)

	partner := &model.RemoteAgent{
		Name: "partner", Protocol: testProtocol,
		Address: types.Addr("localhost", 2222),
	}
	So(db.Insert(partner).Run(), ShouldBeNil)

	remAccount := &model.RemoteAccount{
		RemoteAgentID: partner.ID,
		Login:         "titi",
	}
	So(db.Insert(remAccount).Run(), ShouldBeNil)

	cliService := &protocolstest.TestService{}
	So(cliService.Start(), ShouldBeNil)

	services.Clients[client.Name] = cliService

	Reset(func() { delete(services.Clients, client.Name) })

	return &testContext{
		root:          rootPath,
		db:            db,
		logger:        logger,
		client:        client,
		partner:       partner,
		remoteAccount: remAccount,
		server:        server,
		localAccount:  locAccount,
		send:          send,
		recv:          recv,
	}
}
