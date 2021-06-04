package pipelinetest

import (
	"crypto/rand"
	"encoding/json"
	"io/ioutil"
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

const (
	TestFileSize uint64 = 1000000 // 1MB

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
	makeChan(c)

	return &testData{
		Logger: logger,
		DB:     db,
		Paths:  paths,
	}
}

func makeChan(c convey.C) {
	taskstest.ClientCheckChannel = make(chan string, 20)
	taskstest.ServerCheckChannel = make(chan string, 20)
	c.Reset(func() {
		if taskstest.ClientCheckChannel != nil {
			close(taskstest.ClientCheckChannel)
		}
		if taskstest.ServerCheckChannel != nil {
			close(taskstest.ServerCheckChannel)
		}
		taskstest.ClientCheckChannel = nil
		taskstest.ServerCheckChannel = nil
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
