package database

import (
	"fmt"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	_ "github.com/mattn/go-sqlite3" // register the sqlite driver
	"xorm.io/xorm"
)

const (
	// SQLite is the configuration option for using the SQLite RDBMS
	SQLite = "sqlite"

	// SqliteDriver is the name of the SQLite database driver
	SqliteDriver = "sqlite3"
)

func init() {
	supportedRBMS[SQLite] = sqliteinfo
}

func sqliteInit(db *xorm.Engine) error {
	db.SetMaxOpenConns(1)
	db.DatabaseTZ = time.UTC
	_, _ = db.Exec("PRAGMA busy_timeout = 10000")
	return nil
}

func sqliteinfo() (string, string, func(*xorm.Engine) error) {
	return SqliteDriver, SqliteDSN(), sqliteInit
}

// SqliteDSN takes a database configuration and returns the corresponding
// SQLite DSN necessary to connect to the database.
func SqliteDSN() string {
	config := &conf.GlobalConfig.Database
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
