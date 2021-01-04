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
)

const (
	test       = "test"
	testDriver = "sqlite3"
)

func testinfo(c conf.DatabaseConfig) (string, string) {
	return testDriver, fmt.Sprintf("file:%s?mode=memory&cache=shared&mode=rwc", c.Name)
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

// TestDatabase returns a testing Sqlite database stored in memory for testing
// purposes. The function must be called within a convey context.
// The database will log messages at the level given.
func TestDatabase(c convey.C, logLevel string) *DB {
	supportedRBMS[test] = testinfo
	BcryptRounds = bcrypt.MinCost

	config := &conf.ServerConfig{}
	config.GatewayName = "test_gateway"
	f, err := ioutil.TempFile(os.TempDir(), "test_database_*.db")
	c.So(err, convey.ShouldBeNil)
	c.So(f.Close(), convey.ShouldBeNil)
	c.So(os.Remove(f.Name()), convey.ShouldBeNil)

	config.Database.Name = f.Name()
	config.Database.Type = test

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
	c.Reset(func() {
		_ = db.engine.Close()
		_ = os.Remove(f.Name())
	})
	db.engine.SetMaxOpenConns(1)

	return db
}
