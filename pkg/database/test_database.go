package database

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"
	"xorm.io/xorm"
	"xorm.io/xorm/contexts"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/migrations"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

const (
	testDBEnv       = "GATEWAY_TEST_DB"
	testDBMechanism = "GATEWAY_TEST_DB_MECHANISM"
)

var errSimulated = fmt.Errorf("simulated database error")

func testDSN() string {
	config := conf.GlobalConfig.Database
	values := url.Values{}

	values.Set("mode", "memory")
	values.Set("_txlock", "immediate")
	values.Add("_pragma", "busy_timeout=5000")
	values.Add("_pragma", "foreign_keys=ON")
	values.Add("_pragma", "journal_mode=WAL")
	values.Add("_pragma", "synchronous=NORMAL")

	return fmt.Sprintf("%s?%s", config.Address, values.Encode())
}

func testinfo() (string, string, func(*xorm.Engine) error) {
	return "sqlite", testDSN(), sqliteInit
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
		supportedRBMS[SQLite] = testinfo
		config.Type = SQLite
		config.Address = tempFilename()
	default:
		panic(fmt.Sprintf("Unknown database type '%s'\n", dbType))
	}
}

func resetDB(db *DB) {
	config := &conf.GlobalConfig.Database

	switch config.Type {
	case PostgreSQL, MySQL:
		for _, tbl := range tables {
			convey.So(db.engine.Cascade(true).DropTable(tbl.TableName()), convey.ShouldBeNil)
		}

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

// UnstartedTestDatabase returns an unstarted test database. Starting and
// stopping the service is thus the caller's responsibility.
func UnstartedTestDatabase(c convey.C) *DB {
	db := initTestDatabase(c)

	return db
}

// TestDatabase returns a testing SQLite database stored in memory for testing
// purposes. The function must be called within a convey context.
// The database will log messages at the level given.
func TestDatabase(c convey.C) *DB {
	db := initTestDatabase(c)

	c.So(db.Start(), convey.ShouldBeNil)
	c.Reset(func() { resetDB(db) })

	return db
}

func TestDatabaseNoInit(c convey.C) *DB {
	db := initTestDatabase(c)

	c.So(db.start(false), convey.ShouldBeNil)
	c.Reset(func() { resetDB(db) })

	return db
}

func initTestDatabase(c convey.C) *DB {
	BcryptRounds = bcrypt.MinCost

	initTestDBConf()
	testGCM()

	db := &DB{logger: testhelpers.TestLogger(c, "test_database")}

	if os.Getenv(testDBMechanism) == "migration" {
		startViaMigration(c)
	}

	return db
}

func startViaMigration(c convey.C) {
	config := &conf.GlobalConfig

	var (
		sqlDB   *sql.DB
		dialect string
	)

	switch config.Database.Type {
	case PostgreSQL:
		sqlDB = testhelpers.GetTestPostgreDBNoReset(c)
		dialect = migrations.PostgreSQL

		_, err := sqlDB.Exec(migrations.PostgresCreationScript)
		c.So(err, convey.ShouldBeNil)
	case MySQL:
		sqlDB = testhelpers.GetTestMySQLDBNoReset(c)
		dialect = migrations.MySQL

		script := strings.Split(migrations.MysqlCreationScript, ";\n")
		for _, cmd := range script {
			_, err := sqlDB.Exec(cmd)
			c.So(err, convey.ShouldBeNil)
		}
	case SQLite:
		var addr string
		sqlDB, addr = testhelpers.GetTestSqliteDBNoReset(c)

		dialect = migrations.SQLite
		config.Database.Address = addr

		_, err := sqlDB.Exec(migrations.SqliteCreationScript)
		c.So(err, convey.ShouldBeNil)
	default:
		panic(fmt.Sprintf("Unknown database type '%s'\n", config.Database.Type))
	}

	logger := testhelpers.TestLogger(c, "migration_engine")
	migrations.BumpToCurrent(c, sqlDB, logger, dialect)
	c.So(sqlDB.Close(), convey.ShouldBeNil)
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
