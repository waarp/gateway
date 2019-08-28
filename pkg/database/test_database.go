package database

import (
	"code.bcarlin.xyz/go/logging"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"golang.org/x/crypto/bcrypt"
)

// GetTestDatabase returns a testing Sqlite database stored in memory. If the
// database cannot be started, the function will panic.
func GetTestDatabase() *Db {
	model.BcryptRounds = bcrypt.MinCost

	config := &conf.ServerConfig{}
	config.Database.Type = sqlite
	config.Database.Name = "file::memory:?mode=memory&cache=shared"

	logger := log.NewLogger("test-database")
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
