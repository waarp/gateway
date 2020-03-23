package database

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"code.bcarlin.xyz/go/logging"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"
)

var num uint64 = 0

const (
	test       = "test"
	testDriver = "sqlite3"
)

func init() {
	supportedRBMS[test] = testinfo
}

func testinfo(config conf.DatabaseConfig) (string, string) {
	return testDriver, testDSN(config)
}

func testDSN(config conf.DatabaseConfig) string {
	return fmt.Sprintf("file:%s?mode=memory&cache=shared", config.Name)
}

// GetTestDatabase returns a testing Sqlite database stored in memory. If the
// database cannot be started, the function will panic.
func GetTestDatabase() *DB {
	BcryptRounds = bcrypt.MinCost

	config := &conf.ServerConfig{}
	config.GatewayName = "test_gateway"
	config.Database.Type = test

	name := fmt.Sprint(atomic.LoadUint64(&num))
	convey.Reset(func() { _ = os.Remove(name) })
	config.Database.AESPassphrase = name
	config.Database.Name = name
	atomic.AddUint64(&num, 1)

	db := &DB{
		Conf:       config,
		testDBLock: &sync.Mutex{},
	}
	db.logger = log.NewLogger(ServiceName)
	db.logger.Logger.SetLevel(logging.CRITICAL)
	convey.So(db.Start(), convey.ShouldBeNil)

	return db
}
