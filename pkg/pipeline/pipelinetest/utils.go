package pipelinetest

import (
	"crypto/rand"
	"encoding/json"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"

	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tasks/taskstest"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"
	"github.com/smartystreets/goconvey/convey"
)

// TestFileSize defines the size of the file used for transfer tests.
const TestFileSize int64 = 1000000 // 1MB

// ProgressComplete defines the value a transfer's progress should have at the
// end of the transfert.
const ProgressComplete = uint64(TestFileSize)

// UndefinedProgress is the value used to specify that the transfer's progress
// is undefined at the end of the test (meaning it can have any value between
// 0 and ProgressComplete).
const UndefinedProgress uint64 = math.MaxUint64

// TestLogin and TestPassword are the credentials used for authentication
// during transfer tests.
const (
	TestLogin    = "toto"
	TestPassword = "sesame"
)

type testData struct {
	Logger *log.Logger
	DB     *database.DB
	Paths  *conf.PathsConfig
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
	cont := make([]byte, ProgressComplete)
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

	return &testData{
		Logger: logger,
		DB:     db,
		Paths:  paths,
	}
}

// MakeClientChan initialises the client task checking channel.
func MakeClientChan(c convey.C) {
	setTestVar()
	taskstest.ClientCheckChannel = make(chan string, 20)
	c.Reset(func() {
		taskstest.ClientShouldBeEnd(c)
	})
}

// MakeServerChan initialises the server task checking channel.
func MakeServerChan(c convey.C) {
	setTestVar()
	taskstest.ServerCheckChannel = make(chan string, 20)
	c.Reset(func() {
		taskstest.ServerShouldBeEnd(c)
	})
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

func makeRuleTasks(c convey.C, db *database.DB, rule *model.Rule, isClient bool) {
	taskType := taskstest.ClientOK
	if !isClient {
		taskType = taskstest.ServerOK
	}

	cPreTask := &model.Task{
		RuleID: rule.ID,
		Chain:  model.ChainPre,
		Rank:   0,
		Type:   taskType,
		Args:   json.RawMessage(`{"msg":"PRE-TASKS[0]"}`),
	}
	c.So(db.Insert(cPreTask).Run(), convey.ShouldBeNil)

	cPostTask := &model.Task{
		RuleID: rule.ID,
		Chain:  model.ChainPost,
		Rank:   0,
		Type:   taskType,
		Args:   json.RawMessage(`{"msg":"POST-TASKS[0]"}`),
	}
	c.So(db.Insert(cPostTask).Run(), convey.ShouldBeNil)

	cErrTask := &model.Task{
		RuleID: rule.ID,
		Chain:  model.ChainError,
		Rank:   0,
		Type:   taskType,
		Args:   json.RawMessage(`{"msg":"ERROR-TASKS[0]"}`),
	}
	c.So(db.Insert(cErrTask).Run(), convey.ShouldBeNil)
}
