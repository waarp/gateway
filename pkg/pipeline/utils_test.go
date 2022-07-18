package pipeline

import (
	"encoding/json"
	"errors"
	"io"
	"path"
	"sync/atomic"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/fstest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

type testContext struct {
	root string
	db   *database.DB
	fs   fs.FS

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
	config.ProtoConfigs[testProtocol] = &config.ConfigMaker{
		Server:  func() config.ServerProtoConfig { return new(testhelpers.TestProtoConfig) },
		Partner: func() config.PartnerProtoConfig { return new(testhelpers.TestProtoConfig) },
		Client:  func() config.ClientProtoConfig { return new(testhelpers.TestProtoConfig) },
	}
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
		fs:            testFS,
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

type testPipeline struct {
	*Pipeline
	preTasks,
	postTasks,
	errTasks uint32
	transDone chan bool
}

func newTestPipeline(c C, db *database.DB, trans *model.Transfer) *testPipeline {
	pip, err := NewClientPipeline(db, trans)
	c.So(err, ShouldBeNil)
	pip.Pip.updTicker.Reset(testTransferUpdateInterval)

	testPip := &testPipeline{Pipeline: pip.Pip, transDone: make(chan bool)}

	pip.Pip.Trace = Trace{
		OnPreTask: func(rank int8) error {
			atomic.AddUint32(&testPip.preTasks, 1)

			return nil
		},
		OnPostTask: func(rank int8) error {
			atomic.AddUint32(&testPip.postTasks, 1)

			return nil
		},
		OnErrorTask:   func(rank int8) { atomic.AddUint32(&testPip.errTasks, 1) },
		OnTransferEnd: func() { close(testPip.transDone) },
	}

	return testPip
}

//nolint:gochecknoglobals //this is just for tests
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
