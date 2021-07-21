package database

import (
	"fmt"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	_ "github.com/mattn/go-sqlite3" // register the sqlite driver
	"xorm.io/xorm"
)

const (
	// Configuration option for using the Sqlite RDBMS
	sqlite = "sqlite"

	// SqliteDriver is the name of the Sqlite database driver
	SqliteDriver = "sqlite3"
)

func init() {
	supportedRBMS[sqlite] = sqliteinfo
}

func sqliteInit(db *xorm.Engine) error {
	db.SetMaxOpenConns(1)
	db.DatabaseTZ = time.UTC
	return nil
}

func sqliteinfo(config *conf.DatabaseConfig) (string, string, func(*xorm.Engine) error) {
	return SqliteDriver, SqliteDSN(config), sqliteInit
}

// SqliteDSN takes a database configuration and returns the corresponding
// Sqlite DSN necessary to connect to the database.
func SqliteDSN(config *conf.DatabaseConfig) string {
	var user, pass string
	if config.User != "" {
		user = fmt.Sprintf("&_auth_user=%s", config.User)
	}
	if config.Password != "" {
		pass = fmt.Sprintf("&_auth_pass=%s", config.Password)
	}
	return fmt.Sprintf(
		"file:%s?cache=shared&mode=rwc&_busy_timeout=10000%s%s",
		config.Address, user, pass)
}
