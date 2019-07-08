package database

import (
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	_ "github.com/mattn/go-sqlite3" // register the sqlite driver
)

const (
	// Configuration option for using the Sqlite RDBMS
	sqlite = "sqlite"

	// Name of the Sqlite database driver
	sqliteDriver = "sqlite3"
)

func init() {
	supportedRBMS[sqlite] = sqliteinfo
}

func sqliteinfo(config conf.DatabaseConfig) (string, string) {
	return sqliteDriver, sqliteDSN(config)
}

func sqliteDSN(config conf.DatabaseConfig) string {
	var user, pass string
	if config.User != "" {
		user = fmt.Sprintf("?_auth_user=%s", config.User)
	}
	if config.Password != "" {
		pass = fmt.Sprintf("&_auth_pass=%s", config.Password)
	}
	return fmt.Sprintf("%s%s%s", config.Name, user, pass)
}
