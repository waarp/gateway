package pipelinetest

import (
	"crypto/rand"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/fstest"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks/taskstest"
)

// TestLogin and TestPassword are the credentials used for authentication
// during transfer tests.
const (
	TestLogin    = "foo"
	TestPassword = "sesame"
)

//nolint:gochecknoinits //init is required here
func init() {
	if err := logging.AddLogBackend("TRACE", "stdout", "", ""); err != nil {
		panic(fmt.Sprintf("failed to initialize the log backend: %v", err))
	}
}

// TestFileSize defines the size of the file used for transfer tests.
const TestFileSize int64 = 1000000 // 1MB

var ErrTestError = types.NewTransferError(types.TeInternal, "intended test error")

type testData struct {
	DB    *database.DB
	FS    fs.FS
	Paths *conf.PathsConfig

	hasClientDataError bool
	hasServerDataError bool

	cliPreTasksNb, cliPostTasksNb, cliErrTasksNb    uint32
	servPreTasksNb, servPostTasksNb, servErrTasksNb uint32
	cliDone, servDone                               chan bool
}

func (t *testData) makeServerTracer(isSend bool) func() pipeline.Trace {
	return func() pipeline.Trace {
		trace := pipeline.Trace{
			OnPreTask: func(int8) error {
				atomic.AddUint32(&t.servPreTasksNb, 1)

				return nil
			},
			OnPostTask: func(int8) error {
				atomic.AddUint32(&t.servPostTasksNb, 1)

				return nil
			},
			OnErrorTask: func(int8) {
				atomic.AddUint32(&t.servErrTasksNb, 1)
			},
			OnTransferEnd: func() { close(t.servDone) },
		}

		//nolint:nestif //no easy way to factorize
		if t.hasServerDataError {
			if isSend {
				trace.OnRead = func(off int64) error {
					if off >= DataErrorOffset {
						return ErrTestError
					}

					return nil
				}
			} else {
				trace.OnWrite = func(off int64) error {
					if off >= DataErrorOffset {
						return ErrTestError
					}

					return nil
				}
				trace.OnRead = func(off int64) error {
					if off >= DataErrorOffset {
						<-time.After(200 * time.Millisecond) //nolint:gomnd //for test only
					}

					return nil
				}
			}
		}

		return trace
	}
}

func (t *testData) setClientTrace(pip *pipeline.Pipeline) {
	pip.Trace.OnPreTask = func(int8) error {
		atomic.AddUint32(&t.cliPreTasksNb, 1)

		return nil
	}

	pip.Trace.OnPostTask = func(int8) error {
		atomic.AddUint32(&t.cliPostTasksNb, 1)

		return nil
	}

	pip.Trace.OnErrorTask = func(int8) {
		atomic.AddUint32(&t.cliErrTasksNb, 1)
	}

	pip.Trace.OnTransferEnd = func() { close(t.cliDone) }

	//nolint:nestif //no easy way to factorize
	if t.hasClientDataError {
		if pip.TransCtx.Rule.IsSend {
			pip.Trace.OnRead = func(off int64) error {
				if off >= DataErrorOffset {
					return ErrTestError
				}

				return nil
			}
		} else {
			pip.Trace.OnWrite = func(off int64) error {
				if off >= DataErrorOffset {
					return ErrTestError
				}

				return nil
			}
			pip.Trace.OnRead = func(off int64) error {
				if off >= DataErrorOffset {
					<-time.After(200 * time.Millisecond) //nolint:gomnd //for test only
				}

				return nil
			}
		}
	}
}

func hash(pwd string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.MinCost)
	convey.So(err, convey.ShouldBeNil)

	return string(h)
}

func mkURL(base string, elem ...string) *types.URL {
	url, err := types.ParseURL(base)
	convey.So(err, convey.ShouldBeNil)

	return url.JoinPath(elem...)
}

// AddSourceFile creates a file under the given directory with the given name,
// fills it with random data, and then returns said data.
func AddSourceFile(c convey.C, filesys fs.FS, file *types.URL) []byte {
	c.So(fs.MkdirAll(filesys, file.Dir()), convey.ShouldBeNil)

	cont := make([]byte, TestFileSize)

	_, err := rand.Read(cont)
	c.So(err, convey.ShouldBeNil)

	c.So(fs.WriteFullFile(filesys, file, cont), convey.ShouldBeNil)

	return cont
}

func initTestData(c convey.C) *testData {
	db := database.TestDatabase(c)
	testFS := fstest.InitMemFS(c)

	home := "memory:/gw_home"
	homePath := mkURL(home)

	paths := &conf.PathsConfig{
		GatewayHome:   home,
		DefaultInDir:  "in",
		DefaultOutDir: "out",
		DefaultTmpDir: "tmp",
	}

	c.So(fs.MkdirAll(testFS, homePath), convey.ShouldBeNil)
	c.So(fs.MkdirAll(testFS, homePath.JoinPath(paths.DefaultInDir)), convey.ShouldBeNil)
	c.So(fs.MkdirAll(testFS, homePath.JoinPath(paths.DefaultOutDir)), convey.ShouldBeNil)
	c.So(fs.MkdirAll(testFS, homePath.JoinPath(paths.DefaultTmpDir)), convey.ShouldBeNil)

	conf.GlobalConfig.Paths = *paths

	return &testData{
		DB:       db,
		FS:       testFS,
		Paths:    paths,
		cliDone:  make(chan bool),
		servDone: make(chan bool),
	}
}

func makeRuleTasks(c convey.C, db *database.DB, rule *model.Rule) {
	cPreTask := &model.Task{
		RuleID: rule.ID,
		Chain:  model.ChainPre,
		Rank:   0,
		Type:   taskstest.TaskOK,
	}
	c.So(db.Insert(cPreTask).Run(), convey.ShouldBeNil)

	cPostTask := &model.Task{
		RuleID: rule.ID,
		Chain:  model.ChainPost,
		Rank:   0,
		Type:   taskstest.TaskOK,
	}
	c.So(db.Insert(cPostTask).Run(), convey.ShouldBeNil)

	cErrTask := &model.Task{
		RuleID: rule.ID,
		Chain:  model.ChainError,
		Rank:   0,
		Type:   taskstest.TaskOK,
	}
	c.So(db.Insert(cErrTask).Run(), convey.ShouldBeNil)
}

func (t *testData) ClientShouldHavePreTasked(c convey.C) {
	c.So(t.cliPreTasksNb, convey.ShouldNotBeZeroValue)
}

func (t *testData) ServerShouldHavePreTasked(c convey.C) {
	c.So(t.servPreTasksNb, convey.ShouldNotBeZeroValue)
}

func (t *testData) ClientShouldHavePostTasked(c convey.C) {
	c.So(t.cliPostTasksNb, convey.ShouldNotBeZeroValue)
}

func (t *testData) ServerShouldHavePostTasked(c convey.C) {
	c.So(t.servPostTasksNb, convey.ShouldNotBeZeroValue)
}

func (t *testData) ClientShouldHaveErrorTasked(c convey.C) {
	c.So(t.cliErrTasksNb, convey.ShouldNotBeZeroValue)
}

func (t *testData) ServerShouldHaveErrorTasked(c convey.C) {
	c.So(t.servErrTasksNb, convey.ShouldNotBeZeroValue)
}
