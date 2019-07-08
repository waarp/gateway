package database

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
)

// GetTestDatabase returns a testing Sqlite database stored in memory. If the
// database cannot be started, the function will panic.
func GetTestDatabase() *Db {
	config := &conf.ServerConfig{}
	config.Database.Type = sqlite
	config.Database.Name = "file::memory:?mode=memory&cache=shared"

	db := &Db{Conf: config}
	if err := db.Start(); err != nil {
		panic(err)
	}
	return db
}
