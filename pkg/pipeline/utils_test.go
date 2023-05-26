package pipeline

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"

	"code.waarp.fr/lib/log"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
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

const testProtocol = "test_proto"

//nolint:gochecknoinits // init is used by design
func init() {
	config.ProtoConfigs[testProtocol] = func() config.ProtoConfig {
		return new(testhelpers.TestProtoConfig)
	}

	model.ValidTasks[TaskWait] = &taskWait{}
}

func hash(pwd string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.MinCost)
	So(err, ShouldBeNil)

	return string(h)
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
	db := database.TestDatabase(c)

	conf.GlobalConfig.Paths = conf.PathsConfig{
		GatewayHome:   root,
		DefaultInDir:  "in",
		DefaultOutDir: "out",
		DefaultTmpDir: "work",
	}
	So(os.Mkdir(filepath.Join(root, conf.GlobalConfig.Paths.DefaultInDir), 0o700), ShouldBeNil)
	So(os.Mkdir(filepath.Join(root, conf.GlobalConfig.Paths.DefaultOutDir), 0o700), ShouldBeNil)
	So(os.Mkdir(filepath.Join(root, conf.GlobalConfig.Paths.DefaultTmpDir), 0o700), ShouldBeNil)

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

	So(os.Mkdir(filepath.Join(root, send.LocalDir), 0o700), ShouldBeNil)
	So(os.Mkdir(filepath.Join(root, recv.LocalDir), 0o700), ShouldBeNil)
	So(os.Mkdir(filepath.Join(root, recv.TmpLocalRcvDir), 0o700), ShouldBeNil)

	server := &model.LocalAgent{
		Name:        "server",
		Protocol:    testProtocol,
		ProtoConfig: json.RawMessage(`{}`),
		Address:     "localhost:1111",
	}
	So(db.Insert(server).Run(), ShouldBeNil)

	locAccount := &model.LocalAccount{
		LocalAgentID: server.ID,
		Login:        "toto",
		PasswordHash: hash("sesame"),
	}
	So(db.Insert(locAccount).Run(), ShouldBeNil)

	partner := &model.RemoteAgent{
		Name:        "partner",
		Protocol:    testProtocol,
		ProtoConfig: json.RawMessage(`{}`),
		Address:     "localhost:2222",
	}
	So(db.Insert(partner).Run(), ShouldBeNil)

	remAccount := &model.RemoteAccount{
		RemoteAgentID: partner.ID,
		Login:         "titi",
		Password:      "sesame",
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

func mkRecvTransfer(ctx *testContext, filename string) *model.Transfer {
	So(os.MkdirAll(filepath.Join(ctx.root, ctx.send.LocalDir), 0o700), ShouldBeNil)
	So(os.MkdirAll(filepath.Join(ctx.root, ctx.send.TmpLocalRcvDir), 0o700), ShouldBeNil)

	trans := &model.Transfer{
		RemoteAccountID: utils.NewNullInt64(ctx.remoteAccount.ID),
		LocalPath:       filename,
		RemotePath:      filename,
		RuleID:          ctx.recv.ID,
	}
	So(ctx.db.Insert(trans).Run(), ShouldBeNil)

	return trans
}

func mkSendTransfer(ctx *testContext, filename string) *model.Transfer {
	So(os.MkdirAll(filepath.Join(ctx.root, ctx.send.LocalDir), 0o700), ShouldBeNil)
	So(os.MkdirAll(filepath.Join(ctx.root, ctx.send.TmpLocalRcvDir), 0o700), ShouldBeNil)

	trans := &model.Transfer{
		RemoteAccountID: utils.NewNullInt64(ctx.remoteAccount.ID),
		LocalPath:       filename,
		RemotePath:      filename,
		RuleID:          ctx.send.ID,
	}
	So(ctx.db.Insert(trans).Run(), ShouldBeNil)

	So(os.WriteFile(filepath.Join(ctx.root, ctx.send.LocalDir, filename),
		[]byte("new pipeline content"), 0o700), ShouldBeNil)

	return trans
}

const testTransferUpdateInterval = time.Microsecond

func initFilestream(ctx *testContext, trans *model.Transfer) *fileStream {
	pip, err := NewClientPipeline(ctx.db, trans)
	So(err, ShouldBeNil)

	pip.Pip.updTicker.Reset(testTransferUpdateInterval)
	So(pip.Pip.machine.Transition(statePreTasks), ShouldBeNil)
	So(pip.Pip.machine.Transition(statePreTasksDone), ShouldBeNil)
	So(pip.Pip.machine.Transition(stateDataStart), ShouldBeNil)

	stream, err := newFileStream(pip.Pip, false)
	So(err, ShouldBeNil)
	Reset(func() { _ = stream.file.Close() })

	pip.Pip.Stream = stream

	if pip.Pip.TransCtx.Rule.IsSend {
		So(pip.Pip.machine.Transition(stateReading), ShouldBeNil)
	} else {
		So(pip.Pip.machine.Transition(stateWriting), ShouldBeNil)
	}

	return stream
}

func newTestPipeline(c C, db *database.DB, trans *model.Transfer) *Pipeline {
	InitTester(c)

	pip, err := NewClientPipeline(db, trans)
	So(err, ShouldBeNil)
	pip.Pip.updTicker.Reset(testTransferUpdateInterval)

	return pip.Pip
}

//nolint:gochecknoglobals //this is necessary for testing
var (
	errRequest = types.NewTransferError(types.TeConnection, "request failed")
	errPre     = types.NewTransferError(types.TeExternalOperation, "remote pre-tasks failed")
	errData    = types.NewTransferError(types.TeDataTransfer, "data transfer failed")
	errPost    = types.NewTransferError(types.TeExternalOperation, "remote post-tasks failed")
	errEnd     = types.NewTransferError(types.TeFinalization, "remote transfer finalization failed")
	errChan    = make(chan error, 1)
)

//nolint:gochecknoinits // init is used by design
func init() {
	ClientConstructors[testProtocol] = newTestProtoClient
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
	if _, err := io.Copy(io.Discard, s); err != nil {
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
	if t.end {
		return errEnd
	}

	return nil
}

func (t *testProtoClient) SendError(err *types.TransferError) {
	errChan <- err
}

const TaskWait = "TaskWait"

//nolint:gochecknoglobals //this is only used for tests
var taskChan = make(chan bool)

type taskWait struct{}

// Run executes the dummy task, which will always succeed.
func (t *taskWait) Run(context.Context, map[string]string, *database.DB, *log.Logger, *model.TransferContext) error {
	<-taskChan
	time.Sleep(100 * time.Millisecond)

	return nil
}
