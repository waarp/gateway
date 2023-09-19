package controller

import (
	"path"

	"code.waarp.fr/lib/log"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/fstest"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
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

func mkURL(elem ...string) *types.URL {
	full := path.Join(elem...)

	url, err := types.ParseURL(full)
	So(err, ShouldBeNil)

	return url
}

func hash(pwd string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.MinCost)
	So(err, ShouldBeNil)

	return string(h)
}

type testContext struct {
	root   string
	db     *database.DB
	fs     fs.FS
	logger *log.Logger

	client        *model.Client
	partner       *model.RemoteAgent
	remoteAccount *model.RemoteAccount
	server        *model.LocalAgent
	localAccount  *model.LocalAccount

	send *model.Rule
	recv *model.Rule
}

func initTestDB(c C) *testContext {
	db := database.TestDatabase(c)
	logger := testhelpers.TestLogger(c, "Pipeline test")
	filesys := fstest.InitMemFS(c)

	root := "mem:/new_transfer_stream"
	rootPath := mkURL(root)

	paths := conf.PathsConfig{
		GatewayHome:   root,
		DefaultInDir:  "in",
		DefaultOutDir: "out",
		DefaultTmpDir: "work",
	}
	conf.GlobalConfig.Paths = paths

	So(fs.MkdirAll(filesys, rootPath.JoinPath(paths.DefaultInDir)), ShouldBeNil)
	So(fs.MkdirAll(filesys, rootPath.JoinPath(paths.DefaultOutDir)), ShouldBeNil)
	So(fs.MkdirAll(filesys, rootPath.JoinPath(paths.DefaultTmpDir)), ShouldBeNil)

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

	So(fs.MkdirAll(filesys, rootPath.JoinPath(send.LocalDir)), ShouldBeNil)
	So(fs.MkdirAll(filesys, rootPath.JoinPath(recv.LocalDir)), ShouldBeNil)
	So(fs.MkdirAll(filesys, rootPath.JoinPath(recv.TmpLocalRcvDir)), ShouldBeNil)

	server := &model.LocalAgent{
		Name:     "server",
		Protocol: testProtocol,
		Address:  "localhost:1111",
	}
	So(db.Insert(server).Run(), ShouldBeNil)

	locAccount := &model.LocalAccount{
		LocalAgentID: server.ID,
		Login:        "toto",
		PasswordHash: hash("sesame"),
	}
	So(db.Insert(locAccount).Run(), ShouldBeNil)

	client := &model.Client{
		Name:         "client",
		Protocol:     testProtocol,
		LocalAddress: "127.0.0.1:2000",
	}
	So(db.Insert(client).Run(), ShouldBeNil)

	partner := &model.RemoteAgent{
		Name:     "partner",
		Protocol: testProtocol,
		Address:  "localhost:2222",
	}
	So(db.Insert(partner).Run(), ShouldBeNil)

	remAccount := &model.RemoteAccount{
		RemoteAgentID: partner.ID,
		Login:         "titi",
		Password:      "sesame",
	}
	So(db.Insert(remAccount).Run(), ShouldBeNil)

	cliService := &protocolstest.TestService{}
	So(cliService.Start(), ShouldBeNil)

	services.Clients[client.Name] = cliService

	Reset(func() { delete(services.Clients, client.Name) })

	return &testContext{
		root:          root,
		db:            db,
		fs:            filesys,
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
