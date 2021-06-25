package pipeline

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"
	. "github.com/smartystreets/goconvey/convey"
)

type testContext struct {
	root string
	db   *database.DB

	partner       *model.RemoteAgent
	remoteAccount *model.RemoteAccount
	server        *model.LocalAgent
	localAccount  *model.LocalAccount

	send *model.Rule
	recv *model.Rule
}

func init() {
	_ = log.InitBackend("DEBUG", "stdout", "")
}

func hash(pwd string) []byte {
	h, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.MinCost)
	So(err, ShouldBeNil)
	return h
}

func waitEndTransfer(pip *Pipeline) {
	timeout := time.NewTimer(time.Second * 3)
	ticker := time.NewTicker(time.Millisecond * 100)
	defer func() {
		timeout.Stop()
		ticker.Stop()
	}()

	for {
		select {
		case <-timeout.C:
			panic("test transfer end timeout exceeded")
		case <-ticker.C:
			if pip.machine.HasEnded() {
				return
			}
		}
	}
}

func initTestDB(c C) *testContext {
	root := testhelpers.TempDir(c, "new_transfer_stream")
	db := database.TestDatabase(c, "ERROR")

	db.Conf.Paths = conf.PathsConfig{
		GatewayHome:   root,
		DefaultInDir:  "in",
		DefaultOutDir: "out",
		DefaultTmpDir: "work",
	}
	So(os.Mkdir(filepath.Join(root, db.Conf.Paths.DefaultInDir), 0700), ShouldBeNil)
	So(os.Mkdir(filepath.Join(root, db.Conf.Paths.DefaultOutDir), 0700), ShouldBeNil)
	So(os.Mkdir(filepath.Join(root, db.Conf.Paths.DefaultTmpDir), 0700), ShouldBeNil)

	send := &model.Rule{
		Name:   "send",
		IsSend: true,
		Path:   "/send",
	}
	recv := &model.Rule{
		Name:   "recv",
		IsSend: false,
		Path:   "/recv",
	}
	c.So(db.Insert(recv).Run(), ShouldBeNil)
	c.So(db.Insert(send).Run(), ShouldBeNil)

	server := &model.LocalAgent{
		Name:        "server",
		Protocol:    testhelpers.TestProtocol,
		ProtoConfig: json.RawMessage(`{}`),
		Address:     "localhost:1111",
	}
	So(db.Insert(server).Run(), ShouldBeNil)

	locAccount := &model.LocalAccount{
		LocalAgentID: server.ID,
		Login:        "login",
		PasswordHash: hash("password"),
	}
	So(db.Insert(locAccount).Run(), ShouldBeNil)

	partner := &model.RemoteAgent{
		Name:        "partner",
		Protocol:    testhelpers.TestProtocol,
		ProtoConfig: json.RawMessage(`{}`),
		Address:     "localhost:2222",
	}
	So(db.Insert(partner).Run(), ShouldBeNil)

	remAccount := &model.RemoteAccount{
		RemoteAgentID: partner.ID,
		Login:         "login",
		Password:      "password",
	}
	So(db.Insert(remAccount).Run(), ShouldBeNil)

	return &testContext{
		root:          root,
		db:            db,
		partner:       partner,
		remoteAccount: remAccount,
		server:        server,
		localAccount:  locAccount,
		send:          send,
		recv:          recv,
	}
}

func mkRecvTransfer(ctx *testContext, filename string) *model.TransferContext {
	ctx.recv.LocalDir = "local"
	ctx.recv.LocalTmpDir = "tmp"
	ctx.recv.RemoteDir = "remote"
	So(os.MkdirAll(filepath.Join(ctx.root, ctx.send.LocalDir), 0o700), ShouldBeNil)
	So(os.MkdirAll(filepath.Join(ctx.root, ctx.send.LocalTmpDir), 0o700), ShouldBeNil)

	trans := &model.Transfer{
		IsServer:   false,
		AgentID:    ctx.partner.ID,
		AccountID:  ctx.remoteAccount.ID,
		LocalPath:  filename,
		RemotePath: filename,
		RuleID:     ctx.recv.ID,
	}
	So(ctx.db.Insert(trans).Run(), ShouldBeNil)

	return &model.TransferContext{
		Transfer:      trans,
		Rule:          ctx.recv,
		RemoteAgent:   ctx.partner,
		RemoteAccount: ctx.remoteAccount,
		Paths:         &ctx.db.Conf.Paths,
	}
}

func mkSendTransfer(ctx *testContext, filename string) *model.TransferContext {
	ctx.send.LocalDir = "local"
	ctx.send.LocalTmpDir = "tmp"
	ctx.send.RemoteDir = "remote"
	So(os.MkdirAll(filepath.Join(ctx.root, ctx.send.LocalDir), 0o700), ShouldBeNil)
	So(os.MkdirAll(filepath.Join(ctx.root, ctx.send.LocalTmpDir), 0o700), ShouldBeNil)

	trans := &model.Transfer{
		IsServer:   false,
		AgentID:    ctx.partner.ID,
		AccountID:  ctx.remoteAccount.ID,
		LocalPath:  filename,
		RemotePath: filename,
		RuleID:     ctx.send.ID,
	}
	So(ctx.db.Insert(trans).Run(), ShouldBeNil)

	So(ioutil.WriteFile(filepath.Join(ctx.root, ctx.send.LocalDir, filename),
		[]byte("new pipeline content"), 0o700), ShouldBeNil)

	return &model.TransferContext{
		Transfer:      trans,
		Rule:          ctx.send,
		RemoteAgent:   ctx.partner,
		RemoteAccount: ctx.remoteAccount,
		Paths:         &ctx.db.Conf.Paths,
	}
}

func initFilestream(ctx *testContext, logger *log.Logger, transCtx *model.TransferContext) *fileStream {
	pip, err := newPipeline(ctx.db, logger, transCtx)
	So(err, ShouldBeNil)

	So(pip.machine.Transition("pre-tasks"), ShouldBeNil)
	So(pip.machine.Transition("pre-tasks done"), ShouldBeNil)
	So(pip.machine.Transition("start data"), ShouldBeNil)

	stream, err := newFileStream(pip, time.Nanosecond, false)
	So(err, ShouldBeNil)
	Reset(func() { _ = stream.file.Close() })

	return stream
}

var (
	errRequest = types.NewTransferError(types.TeConnection, "request failed")
	errPre     = types.NewTransferError(types.TeExternalOperation, "remote pre-tasks failed")
	errData    = types.NewTransferError(types.TeDataTransfer, "data transfer failed")
	errPost    = types.NewTransferError(types.TeExternalOperation, "remote post-tasks failed")
	errEnd     = types.NewTransferError(types.TeFinalization, "remote transfer finalization failed")
	errChan    = make(chan error, 1)
	pauseChan  = make(chan bool, 1)
)

func init() {
	ClientConstructors[testhelpers.TestProtocol] = newTestProtoClient
}

func newTestProtoClient(*Pipeline) (Client, *types.TransferError) {
	return &testProtoClient{}, nil
}

type testProtoClient struct {
	request, pre1, pre2, data, post1, post2, end bool
}

func (t *testProtoClient) Request() *types.TransferError {
	if t.request {
		return errRequest
	}
	return nil
}

func (t *testProtoClient) BeginPreTasks() *types.TransferError {
	if t.pre1 {
		return errPre
	}
	return nil
}

func (t *testProtoClient) EndPreTasks() *types.TransferError {
	if t.pre2 {
		return errPre
	}
	return nil
}

func (t *testProtoClient) Data(s DataStream) *types.TransferError {
	if _, err := io.Copy(ioutil.Discard, s); err != nil {
		return errData
	}
	if t.data {
		return errData
	}
	return nil
}

func (t *testProtoClient) BeginPostTasks() *types.TransferError {
	if t.post1 {
		return errPost
	}
	return nil
}

func (t *testProtoClient) EndPostTasks() *types.TransferError {
	if t.post2 {
		return errPost
	}
	return nil
}

func (t *testProtoClient) EndTransfer() *types.TransferError {
	if t.request {
		return errEnd
	}
	return nil
}

func (t *testProtoClient) SendError(err *types.TransferError) {
	errChan <- err
}

func (t *testProtoClient) Pause(context.Context) *types.TransferError {
	pauseChan <- true
	return nil
}
