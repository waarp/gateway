package pipeline

import (
	"errors"
	"path"
	"sync/atomic"
	"time"

	"code.waarp.fr/lib/log"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/fstest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/modeltest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

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

const testProtocol = "test_proto"

//nolint:gochecknoinits // init is used by design
func init() {
	modeltest.AddDummyProtoConfig(testProtocol)
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
	db := database.TestDatabase(c)
	logger := testhelpers.TestLogger(c, "Pipeline test")
	testFS := fstest.InitMemFS(c)

	root := "memory:/new_transfer_stream"
	rootPath := mkURL(root)

	paths := conf.PathsConfig{
		GatewayHome:   root,
		DefaultInDir:  "in",
		DefaultOutDir: "out",
		DefaultTmpDir: "work",
	}
	conf.GlobalConfig.Paths = paths

	So(fs.MkdirAll(testFS, rootPath.JoinPath(paths.DefaultInDir)), ShouldBeNil)
	So(fs.MkdirAll(testFS, rootPath.JoinPath(paths.DefaultOutDir)), ShouldBeNil)
	So(fs.MkdirAll(testFS, rootPath.JoinPath(paths.DefaultTmpDir)), ShouldBeNil)

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

	So(fs.MkdirAll(testFS, rootPath.JoinPath(send.LocalDir)), ShouldBeNil)
	So(fs.MkdirAll(testFS, rootPath.JoinPath(recv.LocalDir)), ShouldBeNil)
	So(fs.MkdirAll(testFS, rootPath.JoinPath(recv.TmpLocalRcvDir)), ShouldBeNil)

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

	return &testContext{
		root:          root,
		db:            db,
		fs:            testFS,
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
	So(fs.MkdirAll(ctx.fs, mkURL(ctx.root, ctx.send.LocalDir)), ShouldBeNil)
	So(fs.MkdirAll(ctx.fs, mkURL(ctx.root, ctx.send.TmpLocalRcvDir)), ShouldBeNil)

	trans := &model.Transfer{
		ClientID:        utils.NewNullInt64(ctx.client.ID),
		RemoteAccountID: utils.NewNullInt64(ctx.remoteAccount.ID),
		SrcFilename:     filename,
		RuleID:          ctx.recv.ID,
	}
	So(ctx.db.Insert(trans).Run(), ShouldBeNil)

	return trans
}

const testTransferFileContent = "new pipeline content"

func mkSendTransfer(ctx *testContext, filename string) *model.Transfer {
	So(fs.MkdirAll(ctx.fs, mkURL(ctx.root, ctx.send.LocalDir)), ShouldBeNil)
	So(fs.MkdirAll(ctx.fs, mkURL(ctx.root, ctx.send.TmpLocalRcvDir)), ShouldBeNil)

	trans := &model.Transfer{
		ClientID:        utils.NewNullInt64(ctx.client.ID),
		RemoteAccountID: utils.NewNullInt64(ctx.remoteAccount.ID),
		SrcFilename:     filename,
		RuleID:          ctx.send.ID,
	}
	So(ctx.db.Insert(trans).Run(), ShouldBeNil)

	So(fs.WriteFullFile(ctx.fs, mkURL(ctx.root, ctx.send.LocalDir, filename),
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

	return fs.WriteFile(t.File, p)
}

func (t *testFile) ReadAt(p []byte, off int64) (n int, err error) {
	if t.err != nil {
		return 0, t.err
	}

	return fs.ReadAtFile(t.File, p, off)
}

func (t *testFile) WriteAt(p []byte, off int64) (n int, err error) {
	if t.err != nil {
		return 0, t.err
	}

	return fs.WriteAtFile(t.File, p, off)
}

func initFilestream(ctx *testContext, trans *model.Transfer) *FileStream {
	transCtx, ctxErr := model.GetTransferContext(ctx.db, ctx.logger, trans)
	So(ctxErr, ShouldBeNil)

	pip, pipErr := NewClientPipeline(ctx.db, ctx.logger, transCtx)
	So(pipErr, ShouldBeNil)

	Reset(pip.doneOK)

	pip.updTicker.Reset(testTransferUpdateInterval)
	So(pip.machine.Transition(statePreTasks), ShouldBeNil)
	So(pip.machine.Transition(statePreTasksDone), ShouldBeNil)
	So(pip.machine.Transition(stateDataStart), ShouldBeNil)

	stream, fileErr := newFileStream(pip, false)
	So(fileErr, ShouldBeNil)
	Reset(func() { _ = stream.file.Close() })

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

	pip, err := NewClientPipeline(db, logger, transCtx)
	c.So(err, ShouldBeNil)
	pip.updTicker.Reset(testTransferUpdateInterval)

	resetPip(pip)

	testPip := &testPipeline{Pipeline: pip, transDone: make(chan bool)}

	pip.Trace = Trace{
		OnPreTask: func(rank int8) error {
			atomic.AddUint32(&testPip.preTasks, 1)

			return nil
		},
		OnPostTask: func(rank int8) error {
			atomic.AddUint32(&testPip.postTasks, 1)

			return nil
		},
		OnErrorTask: func(rank int8) { atomic.AddUint32(&testPip.errTasks, 1) },
		OnTransferEnd: func() {
			close(testPip.transDone)
		},
	}

	return testPip
}
