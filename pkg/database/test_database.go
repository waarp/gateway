package database

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"

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

func testinfo(conf.DatabaseConfig) (string, string) {
	return testDriver, testDSN()
}

func testDSN() string {
	return "file::memory:?cache=shared&mode=rwc"
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

// GetTestDatabase returns a testing Sqlite database stored in memory. If the
// database cannot be started, the function will panic.
func GetTestDatabase() *DB {
	supportedRBMS[test] = testinfo
	BcryptRounds = bcrypt.MinCost

	config := &conf.ServerConfig{}
	config.GatewayName = "test_gateway"
	config.Database.Type = test

	logger := &logging.Logger{}
	logger.SetBackend(logging.NewStdoutBackend())
	logger.SetLevel(logging.WARNING)

	testGCM()
	db := &DB{
		Conf:   config,
		logger: &log.Logger{Logger: logger},
	}

	convey.So(db.Start(), convey.ShouldBeNil)
	convey.Reset(func() { _ = db.engine.Close() })

	return db
}
