package pipelinetest

import (
	"crypto/rand"
	"io/ioutil"
	"os"
	"path/filepath"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"

	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tasks/taskstest"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"
	"github.com/smartystreets/goconvey/convey"
)

// TestLogin and TestPassword are the credentials used for authentication
// during transfer tests.
const (
	TestLogin    = "toto"
	TestPassword = "sesame"
)

// TestFileSize defines the size of the file used for transfer tests.
const TestFileSize int64 = 1000000 // 1MB

type testData struct {
	Logger       *log.Logger
	DB           *database.DB
	Paths        *conf.PathsConfig
	TasksChecker *taskstest.TaskChecker
}

type transData struct {
	ClientTrans *model.Transfer
	fileContent []byte
}

func hash(pwd string) []byte {
	h, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.MinCost)
	convey.So(err, convey.ShouldBeNil)
	return h
}

// AddSourceFile creates a file under the given directory with the given name,
// fills it with random data, and then returns said data.
func AddSourceFile(c convey.C, dir, file string) []byte {
	c.So(os.MkdirAll(dir, 0700), convey.ShouldBeNil)
	cont := make([]byte, TestFileSize)
	_, err := rand.Read(cont)
	c.So(err, convey.ShouldBeNil)
	path := filepath.Join(dir, file)
	c.So(ioutil.WriteFile(path, cont, 0o600), convey.ShouldBeNil)
	return cont
}

func initTestData(c convey.C) *testData {
	logger := log.NewLogger("test_logger")
	db := database.TestDatabase(c, "ERROR")
	home := testhelpers.TempDir(c, "transfer_test")
	paths := makePaths(c, home)
	db.Conf.Paths = *paths
	tasksChecker := taskstest.InitTaskChecker()
	model.ValidTasks[taskstest.TaskOK] = &taskstest.TestTask{TaskChecker: tasksChecker}
	model.ValidTasks[taskstest.TaskErr] = &taskstest.TestTaskError{TaskChecker: tasksChecker}
	pipeline.TestPipelineEnd = func(isServer bool) {
		if isServer {
			tasksChecker.ServerDone()
		} else {
			tasksChecker.ClientDone()
		}
	}

	return &testData{
		Logger:       logger,
		DB:           db,
		Paths:        paths,
		TasksChecker: tasksChecker,
	}
}

func makePaths(c convey.C, home string) *conf.PathsConfig {
	paths := &conf.PathsConfig{
		GatewayHome:   home,
		DefaultInDir:  "in",
		DefaultOutDir: "out",
		DefaultTmpDir: "tmp",
	}
	c.So(os.MkdirAll(filepath.Join(home, paths.DefaultInDir), 0o700), convey.ShouldBeNil)
	c.So(os.MkdirAll(filepath.Join(home, paths.DefaultOutDir), 0o700), convey.ShouldBeNil)
	c.So(os.MkdirAll(filepath.Join(home, paths.DefaultTmpDir), 0o700), convey.ShouldBeNil)
	return paths
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
	c.So(t.TasksChecker.ClientPreTaskNB(), convey.ShouldNotEqual, 0)
}

func (t *testData) ServerShouldHavePreTasked(c convey.C) {
	c.So(t.TasksChecker.ServerPreTaskNB(), convey.ShouldNotEqual, 0)
}

func (t *testData) ClientShouldHavePostTasked(c convey.C) {
	c.So(t.TasksChecker.ClientPostTaskNB(), convey.ShouldNotEqual, 0)
}

func (t *testData) ServerShouldHavePostTasked(c convey.C) {
	c.So(t.TasksChecker.ServerPostTaskNB(), convey.ShouldNotEqual, 0)
}

func (t *testData) ClientShouldHaveErrorTasked(c convey.C) {
	c.So(t.TasksChecker.ClientErrTaskNB(), convey.ShouldNotEqual, 0)
}

func (t *testData) ServerShouldHaveErrorTasked(c convey.C) {
	c.So(t.TasksChecker.ServerErrTaskNB(), convey.ShouldNotEqual, 0)
}
