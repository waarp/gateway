package database

import (
	"io/ioutil"
	"os"

	"code.bcarlin.xyz/go/logging"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"golang.org/x/crypto/bcrypt"
)

// GetTestDatabase returns a testing Sqlite database stored in memory. If the
// database cannot be started, the function will panic.
func GetTestDatabase() *Db {
	BcryptRounds = bcrypt.MinCost

	config := &conf.ServerConfig{}
	config.GatewayName = "test_gateway"
	config.Database.Type = sqlite
	config.Database.AESPassphrase = os.TempDir() + "/aes_passphrase"
	f, err := ioutil.TempFile("", "*.db")
	if err != nil {
		panic(err)
	}
	config.Database.Name = "file:" + f.Name() + "?mode=memory&cache=shared"
	_ = f.Close()

	logger := log.NewLogger("test_database")
	discard, err := logging.NewNoopBackend()
	if err != nil {
		panic(err)
	}
	logger.SetBackend(discard)

	db := &Db{
		Conf:   config,
		Logger: logger,
	}
	if err := db.Start(); err != nil {
		panic(err)
	}
	return db
}
