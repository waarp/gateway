package database

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
	"github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

const (
	TestDBEnv    = "GATEWAY_TEST_DB"
	TestMemoryDB = "memory"
)

var ErrSimulated = errors.New("simulated database error")

//nolint:gochecknoglobals //these a test variables
var (
	testAEAD     = makeTestAEAD()
	setTestRDBMS = sync.OnceFunc(func() {
		SupportedRBMS[SQLite] = memDBInfo
	})
)

func memDBInfo(config *conf.DatabaseConfig) (gorm.Dialector, error) {
	values := url.Values{}

	values.Set("mode", "memory")
	values.Set("_txlock", "immediate")
	values.Add("_pragma", "busy_timeout(10000)")
	values.Add("_pragma", "foreign_keys(ON)")
	values.Add("_pragma", "journal_mode(MEMORY)")
	values.Add("_pragma", "synchronous(OFF)")

	dsn := fmt.Sprintf("file:%s?%s", config.Address, values.Encode())
	db, err := sql.Open(SQLiteDriver, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(1)

	return sqlite.New(sqlite.Config{
		DriverName: SQLiteDriver,
		DSN:        dsn,
		Conn:       db,
	}), nil
}

func makeTestAEAD() cipher.AEAD {
	key := make([]byte, aesKeySize)

	if _, err := rand.Read(key); err != nil {
		panic(err)
	}

	ciph, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	gcm, err := cipher.NewGCM(ciph)
	if err != nil {
		panic(err)
	}

	return gcm
}

func tempFilename() string {
	f, err := os.CreateTemp("", "test_database_*.db")
	convey.So(err, convey.ShouldBeNil)
	convey.So(f.Close(), convey.ShouldBeNil)
	convey.So(os.Remove(f.Name()), convey.ShouldBeNil)

	return f.Name()
}

func initTestDBConf(c convey.C) *conf.ServerConfig {
	config := &conf.ServerConfig{}
	config.GatewayName = uuid.NewString()
	config.NodeID = "test_node"

	config.Paths.FilePerms = 0o600
	config.Paths.DirPerms = 0o700
	config.Paths.GatewayHome = testhelpers.TempDir(c, "default")
	config.Paths.DefaultOutDir = "out"
	config.Paths.DefaultInDir = "in"
	config.Paths.DefaultTmpDir = "tmp"

	dbConfig := &config.Database
	dbType := os.Getenv(TestDBEnv)

	switch dbType {
	case PostgreSQL:
		dbConfig.Type = PostgreSQL
		dbConfig.User = "postgres"
		dbConfig.Password = "postgres"
		dbConfig.Name = "waarp_gateway_test"
		dbConfig.Address = "localhost:5432"
	case MySQL:
		dbConfig.Type = MySQL
		dbConfig.User = "root"
		dbConfig.Name = "waarp_gateway_test"
		dbConfig.Address = "localhost:3306"
	case SQLite:
		dbConfig.Type = SQLite
		dbConfig.Address = tempFilename()
	case TestMemoryDB, "":
		dbConfig.Type = SQLite
		dbConfig.Address = uuid.New().String()
	default:
		panic(fmt.Sprintf("Unknown database type '%s'\n", dbType))
	}

	return config
}

func resetDB(db *DB) {
	config := &db.Config.Database

	if db.engine == nil {
		convey.So(db.initEngine(), convey.ShouldBeNil)
	}

	switch config.Type {
	case PostgreSQL:
		err := db.engine.Exec("DROP SCHEMA IF EXISTS public CASCADE").Error
		convey.So(err, convey.ShouldBeNil)

		err = db.engine.Exec("CREATE SCHEMA public").Error
		convey.So(err, convey.ShouldBeNil)
		convey.So(db.close(), convey.ShouldBeNil)
	case MySQL:
		err := db.engine.Exec("DROP DATABASE IF EXISTS waarp_gateway_test").Error
		convey.So(err, convey.ShouldBeNil)

		err = db.engine.Exec("CREATE DATABASE waarp_gateway_test").Error
		convey.So(err, convey.ShouldBeNil)
		convey.So(db.close(), convey.ShouldBeNil)
	case SQLite:
		convey.So(db.close(), convey.ShouldBeNil)

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
	setTestRDBMS()
	db := initTestDatabase(c)

	c.So(db.Start(), convey.ShouldBeNil)
	c.Reset(func() { resetDB(db) })
	db.Logger.Noticef("%s database started", db.Config.Database.Type)

	return db
}

func initTestDatabase(c convey.C) *DB {
	BcryptRounds = bcrypt.MinCost

	config := initTestDBConf(c)
	dbtype := config.Database.Type
	dbname := config.Database.Name

	if dbname == "" {
		dbname = filepath.Base(config.Database.Address)
	}

	db := &DB{
		Logger: testhelpers.TestLoggerWithLevel(c,
			fmt.Sprintf("%s-database-%s", dbtype, dbname), log.LevelWarning),
		Config: config,
		AEAD:   testAEAD,
	}

	return db
}

// SimulateError adds a database hook which always returns an error to simulate
// a database error for test purposes.
func SimulateError(c convey.C, db *DB) {
	c.Reset(func() { db.engine.Error = nil })

	//nolint:errcheck //error is always non-nil here
	_ = db.engine.AddError(ErrSimulated)
}
