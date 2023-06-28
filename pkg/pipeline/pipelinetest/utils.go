package pipelinetest

import (
	"crypto/rand"
	"fmt"
	"path"

	"code.waarp.fr/lib/log"
	"github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/proto"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/fs/fstest"
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
	if err := conf.InitBackend("TRACE", "stdout", "", ""); err != nil {
		panic(fmt.Sprintf("failed to initialize the log backend: %v", err))
	}
}

type serviceConstructor func(db *database.DB, logger *log.Logger) proto.Service

// TestFileSize defines the size of the file used for transfer tests.
const TestFileSize int64 = 1000000 // 1MB

type testData struct {
	DB    *database.DB
	Paths *conf.PathsConfig
}

func hash(pwd string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.MinCost)
	convey.So(err, convey.ShouldBeNil)

	return string(h)
}

func mkURL(elem ...string) *types.URL {
	full := path.Join(elem...)

	url, err := types.ParseURL(full)
	convey.So(err, convey.ShouldBeNil)

	return url
}

// AddSourceFile creates a file under the given directory with the given name,
// fills it with random data, and then returns said data.
func AddSourceFile(c convey.C, file *types.URL) []byte {
	c.So(fs.MkdirAll(file.Dir()), convey.ShouldBeNil)

	cont := make([]byte, TestFileSize)

	_, err := rand.Read(cont)
	c.So(err, convey.ShouldBeNil)

	c.So(fs.WriteFullFile(file, cont), convey.ShouldBeNil)

	return cont
}

func initTestData(c convey.C) *testData {
	db := database.TestDatabase(c)
	fstest.InitMemFS(c)

	home := "mem:/gw_home"
	homePath := mkURL(home)

	paths := &conf.PathsConfig{
		GatewayHome:   home,
		DefaultInDir:  "in",
		DefaultOutDir: "out",
		DefaultTmpDir: "tmp",
	}

	c.So(fs.MkdirAll(homePath), convey.ShouldBeNil)
	c.So(fs.MkdirAll(homePath.JoinPath(paths.DefaultInDir)), convey.ShouldBeNil)
	c.So(fs.MkdirAll(homePath.JoinPath(paths.DefaultOutDir)), convey.ShouldBeNil)
	c.So(fs.MkdirAll(homePath.JoinPath(paths.DefaultTmpDir)), convey.ShouldBeNil)

	conf.GlobalConfig.Paths = *paths

	pipeline.InitTester(c)

	return &testData{
		DB:    db,
		Paths: paths,
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
	c.So(pipeline.Tester.CliPre != 0, convey.ShouldBeTrue)
}

func (t *testData) ServerShouldHavePreTasked(c convey.C) {
	c.So(pipeline.Tester.ServPre != 0, convey.ShouldBeTrue)
}

func (t *testData) ClientShouldHavePostTasked(c convey.C) {
	c.So(pipeline.Tester.CliPost != 0, convey.ShouldBeTrue)
}

func (t *testData) ServerShouldHavePostTasked(c convey.C) {
	c.So(pipeline.Tester.ServPost != 0, convey.ShouldBeTrue)
}

func (t *testData) ClientShouldHaveErrorTasked(c convey.C) {
	c.So(pipeline.Tester.CliErr != 0, convey.ShouldBeTrue)
}

func (t *testData) ServerShouldHaveErrorTasked(c convey.C) {
	c.So(pipeline.Tester.ServErr != 0, convey.ShouldBeTrue)
}
