package database

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	"code.bcarlin.xyz/go/logging"
	"github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"
	"xorm.io/xorm"
	"xorm.io/xorm/contexts"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/migrations"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/migration"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

const (
	memoryDBType    = "test_db"
	testDBEnv       = "GATEWAY_TEST_DB"
	testDBMechanism = "GATEWAY_TEST_DB_MECHANISM"
)

var errSimulated = fmt.Errorf("simulated database error")

func testinfo(c *conf.DatabaseConfig) (string, string, func(*xorm.Engine) error) {
	return "sqlite3", fmt.Sprintf("file:%s?mode=memory&_busy_timeout=10000",
		c.Address), sqliteInit
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
	f, err := ioutil.TempFile(os.TempDir(), "test_database_*.db")
	convey.So(err, convey.ShouldBeNil)
	convey.So(f.Close(), convey.ShouldBeNil)
	convey.So(os.Remove(f.Name()), convey.ShouldBeNil)

	return f.Name()
}

func initTestDBConf(config *conf.DatabaseConfig) {
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
		supportedRBMS[memoryDBType] = testinfo
		config.Type = memoryDBType
		config.Address = tempFilename()
	default:
		panic(fmt.Sprintf("Unknown database type '%s'\n", memoryDBType))
	}
}

func resetDB(db *DB) {
	config := &db.Conf.Database

	switch config.Type {
	case PostgreSQL, MySQL:
		for _, tbl := range tables {
			convey.So(db.engine.Cascade(true).DropTable(tbl.TableName()), convey.ShouldBeNil)
		}

		convey.So(db.engine.Close(), convey.ShouldBeNil)
	case memoryDBType, SQLite:
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
func TestDatabase(c convey.C, logLevel string) *DB {
	db := initTestDatabase(c, logLevel)

	c.So(db.Start(), convey.ShouldBeNil)
	c.Reset(func() { resetDB(db) })

	return db
}

func TestDatabaseNoInit(c convey.C, logLevel string) *DB {
	db := initTestDatabase(c, logLevel)

	c.So(db.start(false), convey.ShouldBeNil)
	c.Reset(func() { resetDB(db) })

	return db
}

func initTestDatabase(c convey.C, logLevel string) *DB {
	BcryptRounds = bcrypt.MinCost
	config := &conf.ServerConfig{}
	config.GatewayName = "test_gateway"
	initTestDBConf(&config.Database)

	level, err := logging.LevelByName(logLevel)
	c.So(err, convey.ShouldBeNil)

	logger := logging.NewLogger("Test Database")
	logger.SetBackend(logging.NewStdoutBackend())
	logger.SetLevel(level)

	testGCM()

	db := &DB{
		Conf:   config,
		logger: &log.Logger{Logger: logger},
	}

	if os.Getenv(testDBMechanism) == "migration" {
		startViaMigration(c, config)
	}

	return db
}

func startViaMigration(c convey.C, config *conf.ServerConfig) {
	Owner = config.GatewayName

	var (
		sqlDB   *sql.DB
		dialect string
	)

	switch config.Database.Type {
	case PostgreSQL:
		sqlDB = testhelpers.GetTestPostgreDBNoReset(c)
		dialect = migration.PostgreSQL

		_, err := sqlDB.Exec(migrations.PostgresCreationScript)
		c.So(err, convey.ShouldBeNil)
	case MySQL:
		sqlDB = testhelpers.GetTestMySQLDBNoReset(c)
		dialect = migration.MySQL

		script := strings.Split(migrations.MysqlCreationScript, ";\n")
		for _, cmd := range script {
			_, err := sqlDB.Exec(cmd)
			c.So(err, convey.ShouldBeNil)
		}
	case SQLite, memoryDBType:
		var addr string
		sqlDB, addr = testhelpers.GetTestSqliteDBNoReset(c)

		dialect = migration.SQLite
		config.Database.Address = addr

		_, err := sqlDB.Exec(migrations.SqliteCreationScript)
		c.So(err, convey.ShouldBeNil)
	default:
		panic(fmt.Sprintf("Unknown database type '%s'\n", config.Database.Type))
	}

	migrEngine, err := migration.NewEngine(sqlDB, dialect, nil)
	c.So(err, convey.ShouldBeNil)

	c.So(migrEngine.Upgrade(migrations.Migrations), convey.ShouldBeNil)
	c.So(migrEngine.Upgrade(migrations.BumpToCurrent()), convey.ShouldBeNil)
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
