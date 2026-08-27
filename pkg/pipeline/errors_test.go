package pipeline

import (
	"errors"
	"path"
	"sync/atomic"
	"testing"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/logtest"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/modeltest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

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

const testProtocol = "test_proto"

//nolint:gochecknoinits // init is used by design
func init() {
	modeltest.AddDummyProtoConfig(testProtocol)
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

func initTestDB(c C, root string) *testContext {
	db := database.TestDatabase(c)
	logger := testhelpers.TestLogger(c, "Pipeline test")

	return makeTransferCtx(db, logger, root, func(err error) {
		So(err, ShouldBeNil)
	})
}

func initTransferCtx(tb testing.TB) *testContext {
	db := dbtest.TestDatabase(tb)
	logger := logtest.GetTestLogger(tb)
	root := tb.TempDir()

	return makeTransferCtx(db, logger, root, func(err error) {
		require.NoError(tb, err)
	})
}

func makeTransferCtx(db *database.DB, logger *log.Logger, root string, assertNoError func(error),
) *testContext {
	paths := conf.PathsConfig{
		GatewayHome:   root,
		DefaultInDir:  "in",
		DefaultOutDir: "out",
		DefaultTmpDir: "work",
		FilePerms:     0o600,
		DirPerms:      0o700,
	}
	db.Config.Paths = paths

	assertNoError(fs.MkdirAll(path.Join(root, paths.DefaultInDir)))
	assertNoError(fs.MkdirAll(path.Join(root, paths.DefaultOutDir)))
	assertNoError(fs.MkdirAll(path.Join(root, paths.DefaultTmpDir)))

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
	assertNoError(db.Insert(recv).Run())
	assertNoError(db.Insert(send).Run())

	assertNoError(fs.MkdirAll(path.Join(root, send.LocalDir)))
	assertNoError(fs.MkdirAll(path.Join(root, recv.LocalDir)))
	assertNoError(fs.MkdirAll(path.Join(root, recv.TmpLocalRcvDir)))

	server := &model.LocalAgent{
		Name: "server", Protocol: testProtocol,
		Address: types.Addr("localhost", 1111),
	}
	assertNoError(db.Insert(server).Run())

	locAccount := &model.LocalAccount{
		LocalAgentID: server.ID,
		Login:        "toto",
	}
	assertNoError(db.Insert(locAccount).Run())

	client := &model.Client{
		Name: "client", Protocol: testProtocol,
		LocalAddress: types.Addr("127.0.0.1", 2000),
	}
	assertNoError(db.Insert(client).Run())

	partner := &model.RemoteAgent{
		Name: "partner", Protocol: testProtocol,
		Address: types.Addr("localhost", 2222),
	}
	assertNoError(db.Insert(partner).Run())

	remAccount := &model.RemoteAccount{
		RemoteAgentID: partner.ID,
		Login:         "titi",
	}
	assertNoError(db.Insert(remAccount).Run())

	return &testContext{
		root:          root,
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

func mkRecvTransfer(ctx *testContext, filename string) *model.Transfer {
	So(fs.MkdirAll(path.Join(ctx.root, ctx.send.LocalDir)), ShouldBeNil)
	So(fs.MkdirAll(path.Join(ctx.root, ctx.send.TmpLocalRcvDir)), ShouldBeNil)

	trans := &model.Transfer{
		ClientID:        ctx.client.NullableID(),
		RemoteAccountID: ctx.remoteAccount.NullableID(),
		SrcFilename:     filename,
		RuleID:          ctx.recv.ID,
	}
	So(ctx.db.Insert(trans).Run(), ShouldBeNil)

	return trans
}

const testTransferFileContent = "new pipeline content"

func mkSendTransfer(ctx *testContext, filename string) *model.Transfer {
	So(fs.MkdirAll(fs.JoinPath(ctx.root, ctx.send.LocalDir)), ShouldBeNil)
	So(fs.MkdirAll(fs.JoinPath(ctx.root, ctx.send.TmpLocalRcvDir)), ShouldBeNil)

	trans := &model.Transfer{
		ClientID:        ctx.client.NullableID(),
		RemoteAccountID: ctx.remoteAccount.NullableID(),
		SrcFilename:     filename,
		RuleID:          ctx.send.ID,
	}
	So(ctx.db.Insert(trans).Run(), ShouldBeNil)

	So(fs.WriteFullFile(path.Join(ctx.root, ctx.send.LocalDir, filename),
		[]byte(testTransferFileContent)), ShouldBeNil)

	return trans
}

const testTransferUpdateInterval = time.Microsecond

func addFileError(stream *FileStream) {
	stream.file = &testFile{File: stream.file, err: errFileTest}
}

var errFileTest = errors.New("intended file error")

type testFile struct {
	fs.File
	err error
}

func (t *testFile) Read(p []byte) (n int, err error) {
	if t.err != nil {
		return 0, t.err
	}

	return t.File.Read(p)
}

func (t *testFile) Write(p []byte) (n int, err error) {
	if t.err != nil {
		return 0, t.err
	}

	return t.File.Write(p)
}

func (t *testFile) ReadAt(p []byte, off int64) (n int, err error) {
	if t.err != nil {
		return 0, t.err
	}

	return t.File.ReadAt(p, off)
}

func (t *testFile) WriteAt(p []byte, off int64) (n int, err error) {
	if t.err != nil {
		return 0, t.err
	}

	return t.File.WriteAt(p, off)
}

func initFilestream(ctx *testContext, trans *model.Transfer) *FileStream {
	transCtx, ctxErr := model.GetTransferContext(ctx.db, ctx.logger, trans)
	So(ctxErr, ShouldBeNil)

	pip, pipErr := NewClientPipeline(ctx.db, ctx.logger, transCtx, nil)
	So(pipErr, ShouldBeNil)

	Reset(pip.doneOK)

	pip.updTicker.Reset(testTransferUpdateInterval)
	So(pip.machine.Transition(statePreTasks), ShouldBeNil)
	So(pip.machine.Transition(statePreTasksDone), ShouldBeNil)
	So(pip.machine.Transition(stateDataStart), ShouldBeNil)

	stream, fileErr := newFileStream(pip, false)
	So(fileErr, ShouldBeNil)
	Reset(func() {
		_ = stream.file.Close()
	})

	pip.Stream = stream

	if pip.TransCtx.Rule.IsSend {
		So(pip.machine.Transition(stateReading), ShouldBeNil)
	} else {
		So(pip.machine.Transition(stateWriting), ShouldBeNil)
	}

	return stream
}

type testPipeline struct {
	*Pipeline
	preTasks,
	postTasks,
	errTasks uint32
	transDone chan bool
}

func newTestPipeline(c C, db *database.DB, trans *model.Transfer) *testPipeline {
	logger := testhelpers.TestLogger(c, "Test client pipeline")
	transCtx, err := model.GetTransferContext(db, logger, trans)
	So(err, ShouldBeNil)

	pip, err := NewClientPipeline(db, logger, transCtx, nil)
	c.So(err, ShouldBeNil)
	pip.updTicker.Reset(testTransferUpdateInterval)

	resetPip(pip)

	testPip := &testPipeline{Pipeline: pip, transDone: make(chan bool)}

	pip.Trace = Trace{
		OnPreTask: func(rank int) error {
			atomic.AddUint32(&testPip.preTasks, 1)

			return nil
		},
		OnPostTask: func(rank int) error {
			atomic.AddUint32(&testPip.postTasks, 1)

			return nil
		},
		OnErrorTask: func(rank int) { atomic.AddUint32(&testPip.errTasks, 1) },
		OnTransferEnd: func() {
			close(testPip.transDone)
		},
	}

	return testPip
}
