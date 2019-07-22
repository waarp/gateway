package database

import (
	"code.bcarlin.xyz/go/logging"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
)

// GetTestDatabase returns a testing Sqlite database stored in memory. If the
// database cannot be started, the function will panic.
func GetTestDatabase() *Db {
	config := &conf.ServerConfig{}
	config.Database.Type = sqlite
	config.Database.Name = "file::memory:?mode=memory&cache=shared"

	db := &Db{
		Conf:   config,
		Logger: log.NewLogger("test-database"),
	}
	if err := db.Logger.SetLevel(logging.ERROR.Name()); err != nil {
		panic(err)
	}
	if err := db.Start(); err != nil {
		panic(err)
	}
	return db
}
