package database

import (
	"fmt"
	"os"
	"sync/atomic"

	"code.bcarlin.xyz/go/logging"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"
)

var num uint64 = 0

// GetTestDatabase returns a testing Sqlite database stored in memory. If the
// database cannot be started, the function will panic.
func GetTestDatabase() *Db {
	BcryptRounds = bcrypt.MinCost

	config := &conf.ServerConfig{}
	config.GatewayName = "test_gateway"
	config.Database.Type = sqlite

	name := fmt.Sprint(atomic.LoadUint64(&num))
	convey.Reset(func() { _ = os.Remove(name) })
	config.Database.AESPassphrase = name

	config.Database.Name = fmt.Sprintf("file:%s?mode=memory&cache=shared", name)
	atomic.AddUint64(&num, 1)

	logger := log.NewLogger("test_database")
	discard, err := logging.NewNoopBackend()
	convey.So(err, convey.ShouldBeNil)

	logger.SetBackend(discard)
	db := &Db{
		Conf:   config,
		Logger: logger,
	}
	convey.So(db.Start(), convey.ShouldBeNil)

	return db
}
