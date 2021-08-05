package database

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"code.bcarlin.xyz/go/logging"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"
	"xorm.io/xorm"
	"xorm.io/xorm/contexts"
)

const (
	testDBType = "test_db"
	testDBEnv  = "GATEWAY_TEST_DB"
)

func testinfo() (string, string, func(*xorm.Engine) error) {
	return "sqlite3", fmt.Sprintf("file:%s?mode=memory&cache=shared&mode=rwc",
		conf.GlobalConfig.ServerConf.Database.Address), sqliteInit
}

func testGCM() {
	if GCM != nil {
		return
	}

	key := make([]byte, 32)
	_, err := rand.Read(key)
	convey.So(err, convey.ShouldBeNil)
	ciph, err := aes.NewCipher(key)
	convey.So(err, convey.ShouldBeNil)

	GCM, err = cipher.NewGCM(ciph)
	convey.So(err, convey.ShouldBeNil)
}

func tempFilename() string {
	f, err := ioutil.TempFile(os.TempDir(), "test_database_*.db")
	convey.So(err, convey.ShouldBeNil)
	convey.So(f.Close(), convey.ShouldBeNil)
	convey.So(os.Remove(f.Name()), convey.ShouldBeNil)
	return f.Name()
}

func initTestDBConf() {
	conf.GlobalConfig.ServerConf.GatewayName = "test_gateway"
	config := &conf.GlobalConfig.ServerConf.Database
	dbType := os.Getenv(testDBEnv)
	switch dbType {
	case PostgreSQL:
		config.Type = PostgreSQL
		config.User = "postgres"
		config.Name = "waarp_gateway_test"
		config.Address = "localhost:5432"
	case MySQL:
		config.Type = MySQL
		config.User = "root"
		config.Name = "waarp_gateway_test"
		config.Address = "localhost:3306"
	case SQLite:
		config.Type = SQLite
		config.Address = tempFilename()
	case "":
		supportedRBMS[testDBType] = testinfo
		config.Type = testDBType
		config.Address = tempFilename()
	default:
		panic(fmt.Sprintf("Unknown database type '%s'\n", dbType))
	}
}

func resetDB(db *DB) {
	config := &conf.GlobalConfig.ServerConf.Database
	switch config.Type {
	case PostgreSQL, MySQL:
		for _, tbl := range tables {
			convey.So(db.engine.DropTables(tbl.TableName()), convey.ShouldBeNil)
		}
		convey.So(db.engine.Close(), convey.ShouldBeNil)
	case testDBType:
		convey.So(db.engine.Close(), convey.ShouldBeNil)
		convey.So(db.engine.Close(), convey.ShouldBeNil)
	case SQLite:
		convey.So(db.engine.Close(), convey.ShouldBeNil)
		convey.So(db.engine.Close(), convey.ShouldBeNil)
		convey.So(os.Remove(config.Address), convey.ShouldBeNil)
	default:
		panic(fmt.Sprintf("Unknown database type '%s'\n", config.Type))
	}
}

// TestDatabase returns a testing SQLite database stored in memory for testing
// purposes. The function must be called within a convey context.
// The database will log messages at the level given.
func TestDatabase(c convey.C, logLevel string) *DB {
	BcryptRounds = bcrypt.MinCost
	initTestDBConf()

	level, err := logging.LevelByName(logLevel)
	c.So(err, convey.ShouldBeNil)

	logger := logging.NewLogger("Test Database")
	logger.SetBackend(logging.NewStdoutBackend())
	logger.SetLevel(level)

	testGCM()
	db := &DB{logger: &log.Logger{Logger: logger}}

	c.So(db.Start(), convey.ShouldBeNil)
	c.Reset(func() { resetDB(db) })

	return db
}

type errHook struct{ once sync.Once }

func (e *errHook) BeforeProcess(c *contexts.ContextHook) (context.Context, error) {
	ctx := c.Ctx
	var err error
	e.once.Do(func() {
		err = fmt.Errorf("simulated database error")
	})
	return ctx, err
}

func (*errHook) AfterProcess(*contexts.ContextHook) error { return nil }

// SimulateError adds a database hook which always returns an error to simulate
// a database error for test purposes.
func SimulateError(_ convey.C, db *DB) {
	db.engine.AddHook(&errHook{})
}
