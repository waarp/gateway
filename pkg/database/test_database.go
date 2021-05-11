package database

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"

	"code.bcarlin.xyz/go/logging"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"
	"xorm.io/xorm"
)

const (
	testDBType = "test"
	testDBEnv  = "GATEWAY_TEST_DB"
)

func testinfo(c conf.DatabaseConfig) (string, string, func(*xorm.Engine) error) {
	return "sqlite3", fmt.Sprintf("file:%s?mode=memory&cache=shared&mode=rwc",
		c.Address), sqliteInit
}

func testGCM() {
	if GCM != nil {
		return
	}

	key := make([]byte, 32)
	_, err := rand.Read(key)
	convey.So(err, convey.ShouldBeNil)
	c, err := aes.NewCipher(key)
	convey.So(err, convey.ShouldBeNil)

	GCM, err = cipher.NewGCM(c)
	convey.So(err, convey.ShouldBeNil)
}

func tempFilename(c convey.C) string {
	f, err := ioutil.TempFile(os.TempDir(), "test_database_*.db")
	c.So(err, convey.ShouldBeNil)
	c.So(f.Close(), convey.ShouldBeNil)
	c.So(os.Remove(f.Name()), convey.ShouldBeNil)
	return f.Name()
}

func initTestDBConf(c convey.C, config *conf.DatabaseConfig) {
	dbType := os.Getenv(testDBEnv)
	switch dbType {
	case postgres:
		config.Type = postgres
		config.User = "postgres"
		config.Name = "waarp_gateway_test"
		config.Address = "localhost:5432"
	case mysql:
		config.Type = mysql
		config.User = "root"
		config.Name = "waarp_gateway_test"
		config.Address = "localhost:3306"
	case sqlite:
		config.Type = sqlite
		config.Address = tempFilename(c)
	case "":
		supportedRBMS[testDBType] = testinfo
		config.Type = testDBType
		config.Address = tempFilename(c)
	default:
		_, _ = c.Printf("Unknown database type '%s'\n", dbType)
		c.So(dbType, convey.ShouldNotEqual, dbType)
	}
}

func resetDB(c convey.C, db *DB, config *conf.DatabaseConfig) {
	switch config.Type {
	case postgres, mysql:
		for _, tbl := range tables {
			c.So(db.engine.DropTables(tbl.TableName()), convey.ShouldBeNil)
		}
		c.So(db.engine.Close(), convey.ShouldBeNil)
	case sqlite, testDBType:
		c.So(db.engine.Close(), convey.ShouldBeNil)
		c.So(db.engine.Close(), convey.ShouldBeNil)
		c.So(os.Remove(config.Address), convey.ShouldBeNil)
	default:
		_, _ = c.Printf("Unknown database type '%s'\n", config.Type)
		c.So(config.Type, convey.ShouldNotEqual, config.Type)
	}
}

// TestDatabase returns a testing Sqlite database stored in memory for testing
// purposes. The function must be called within a convey context.
// The database will log messages at the level given.
func TestDatabase(c convey.C, logLevel string) *DB {
	BcryptRounds = bcrypt.MinCost
	config := &conf.ServerConfig{}
	config.GatewayName = "test_gateway"
	initTestDBConf(c, &config.Database)

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

	c.So(db.Start(), convey.ShouldBeNil)
	c.Reset(func() { resetDB(c, db, &config.Database) })

	return db
}
