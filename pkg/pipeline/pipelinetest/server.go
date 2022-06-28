package pipelinetest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"code.waarp.fr/lib/r66"
	"github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/service"
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

func initServer(c convey.C, proto string, constr serviceConstructor,
	servConf config.ProtoConfig,
) *ServerContext {
	t := initTestData(c)
	port := testhelpers.GetFreePort(c)
	server, locAcc := makeServerConf(c, t.DB, port, t.Paths.GatewayHome, proto, servConf)

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
func InitServerPush(c convey.C, proto string, constr serviceConstructor,
	servConf config.ProtoConfig,
) *ServerContext {
	ctx := initServer(c, proto, constr, servConf)
	ctx.ServerRule = makeServerPush(c, ctx.DB)

	return ctx
}

// InitServerPull creates a database and fills it with all the elements necessary
// for a server pull transfer test of the given protocol. It then returns all these
// element inside a ServerContext.
func InitServerPull(c convey.C, proto string, constr serviceConstructor,
	servConf config.ProtoConfig,
) *ServerContext {
	ctx := initServer(c, proto, constr, servConf)
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

func makeServerConf(c convey.C, db *database.DB, port uint16, home, proto string,
	servConf config.ProtoConfig,
) (ag *model.LocalAgent, acc *model.LocalAccount) {
	jsonServConf := json.RawMessage(`{}`)

	if servConf != nil {
		var err error
		jsonServConf, err = json.Marshal(servConf)
		c.So(err, convey.ShouldBeNil)
	}

	root := filepath.Join(home, proto+"_server_root")
	c.So(os.MkdirAll(root, 0o700), convey.ShouldBeNil)

	server := &model.LocalAgent{
		Name:          "server",
		Protocol:      proto,
		RootDir:       utils.ToOSPath(root),
		ProtoConfig:   jsonServConf,
		Address:       fmt.Sprintf("127.0.0.1:%d", port),
		ReceiveDir:    "receive",
		SendDir:       "send",
		TmpReceiveDir: "tmp",
	}

	c.So(db.Insert(server).Run(), convey.ShouldBeNil)
	c.So(os.MkdirAll(filepath.Join(root, server.ReceiveDir), 0o700), convey.ShouldBeNil)
	c.So(os.MkdirAll(filepath.Join(root, server.SendDir), 0o700), convey.ShouldBeNil)
	c.So(os.MkdirAll(filepath.Join(root, server.TmpReceiveDir), 0o700), convey.ShouldBeNil)

	pswd := TestPassword
	if proto == "r66" {
		pswd = string(r66.CryptPass([]byte(pswd)))
	}

	locAccount := &model.LocalAccount{
		LocalAgentID: server.ID,
		Login:        TestLogin,
		PasswordHash: hash(pswd),
	}

	c.So(db.Insert(locAccount).Run(), convey.ShouldBeNil)

	return server, locAccount
}

// AddCryptos adds the given cryptos to the test database.
func (s *ServerContext) AddCryptos(c convey.C, certs ...model.Crypto) {
	for i := range certs {
		c.So(s.DB.Insert(&certs[i]).Run(), convey.ShouldBeNil)
	}
}

// StartService starts the service associated with the server defined in ServerContext.
func (s *ServerContext) StartService(c convey.C) service.Service {
	logger := log.NewLogger(fmt.Sprintf("test_%s_server", s.Server.Protocol))
	serv := s.constr(s.DB, s.Server, logger)
	c.So(serv.Start(), convey.ShouldBeNil)
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
	progress := uint64(TestFileSize)
	s.checkServerTransferOK(c, remoteID, s.filename, progress, s.DB, &actual)
}
