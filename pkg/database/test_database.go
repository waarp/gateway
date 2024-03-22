package database

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"code.waarp.fr/lib/log"
	"github.com/google/uuid"
	"github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"
	"xorm.io/xorm/contexts"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/migrations"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

const (
	TestDBEnv    = "GATEWAY_TEST_DB"
	TestMemoryDB = "memory"
)

var errSimulated = errors.New("simulated database error")

func memDBInfo() *DBInfo {
	config := conf.GlobalConfig.Database
	values := url.Values{}

	values.Set("mode", "memory")
	values.Set("cache", "shared")
	values.Set("_txlock", "immediate")
	values.Add("_pragma", "busy_timeout(10000)")
	values.Add("_pragma", "foreign_keys(ON)")
	values.Add("_pragma", "journal_mode(MEMORY)")
	values.Add("_pragma", "synchronous(OFF)")

	return &DBInfo{
		Driver:    migrations.SqliteDriver,
		DSN:       fmt.Sprintf("file:%s?%s", config.Address, values.Encode()),
		ConnLimit: 1,
	}
}

func testGCM() {
	if GCM != nil {
		return
	}

	key := make([]byte, aesKeySize)

	_, err := rand.Read(key)
	convey.So(err, convey.ShouldBeNil)

	ciph, err := aes.NewCipher(key)
	convey.So(err, convey.ShouldBeNil)

	GCM, err = cipher.NewGCM(ciph)
	convey.So(err, convey.ShouldBeNil)
}

func tempFilename() string {
	f, err := os.CreateTemp("", "test_database_*.db")
	convey.So(err, convey.ShouldBeNil)
	convey.So(f.Close(), convey.ShouldBeNil)
	convey.So(os.Remove(f.Name()), convey.ShouldBeNil)

	return f.Name()
}

func initTestDBConf() {
	BcryptRounds = bcrypt.MinCost
	conf.GlobalConfig.GatewayName = "test_gateway"
	conf.GlobalConfig.NodeID = "test_node"
	config := &conf.GlobalConfig.Database
	dbType := os.Getenv(TestDBEnv)

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
	case TestMemoryDB, "":
		SupportedRBMS[SQLite] = memDBInfo
		config.Type = SQLite
		config.Address = uuid.New().String()
	default:
		panic(fmt.Sprintf("Unknown database type '%s'\n", dbType))
	}
}

func resetDB(db *DB) {
	config := &conf.GlobalConfig.Database

	switch config.Type {
	case PostgreSQL:
		_, err := db.engine.Exec("DROP SCHEMA IF EXISTS public CASCADE")
		convey.So(err, convey.ShouldBeNil)

		_, err = db.engine.Exec("CREATE SCHEMA public")
		convey.So(err, convey.ShouldBeNil)
		convey.So(db.engine.Close(), convey.ShouldBeNil)
	case MySQL:
		_, err := db.engine.Exec("DROP DATABASE IF EXISTS waarp_gateway_test")
		convey.So(err, convey.ShouldBeNil)

		_, err = db.engine.Exec("CREATE DATABASE waarp_gateway_test")
		convey.So(err, convey.ShouldBeNil)
		convey.So(db.engine.Close(), convey.ShouldBeNil)
	case SQLite:
		convey.So(db.engine.Close(), convey.ShouldBeNil)

		if _, err := os.Stat(config.Address); err == nil {
			convey.So(os.Remove(config.Address), convey.ShouldBeNil)
		}
	default:
		panic(fmt.Sprintf("Unknown database type '%s'\n", config.Type))
	}
}

// TestDatabase returns a testing SQLite database stored in memory for testing
// purposes. The function must be called within a convey context.
// The database will log messages at the level given.
func TestDatabase(c convey.C) *DB {
	db := initTestDatabase(c)

	c.So(db.Start(), convey.ShouldBeNil)
	c.Reset(func() { resetDB(db) })
	db.logger.Notice("%s database started", conf.GlobalConfig.Database.Type)

	return db
}

func initTestDatabase(c convey.C) *DB {
	BcryptRounds = bcrypt.MinCost

	initTestDBConf()
	testGCM()

	dbname := conf.GlobalConfig.Database.Name
	if dbname == "" {
		dbname = filepath.Base(conf.GlobalConfig.Database.Address)
	}

	db := &DB{logger: testhelpers.TestLoggerWithLevel(c, dbname, log.LevelNotice)}

	return db
}

type errHook struct{ once sync.Once }

func (e *errHook) BeforeProcess(c *contexts.ContextHook) (context.Context, error) {
	var err error

	ctx := c.Ctx

	e.once.Do(func() {
		err = errSimulated
	})

	return ctx, err
}

func (*errHook) AfterProcess(*contexts.ContextHook) error { return nil }

// SimulateError adds a database hook which always returns an error to simulate
// a database error for test purposes.
func SimulateError(_ convey.C, db *DB) {
	db.engine.AddHook(&errHook{})
}
