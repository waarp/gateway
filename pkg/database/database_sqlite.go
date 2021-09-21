package database

import (
	"fmt"
	"time"

	// Register the sqlite driver.
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
)

const (
	// SQLite is the configuration option for using the SQLite RDBMS.
	SQLite = "sqlite"

	// SqliteDriver is the name of the SQLite database driver.
	SqliteDriver = "sqlite3"
)

//nolint:gochecknoinits // init is used by design
func init() {
	supportedRBMS[SQLite] = sqliteinfo
}

func sqliteInit(db *xorm.Engine) error {
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.DatabaseTZ = time.UTC

	return nil
}

func sqliteinfo(config *conf.DatabaseConfig) (string, string, func(*xorm.Engine) error) {
	return SqliteDriver, SqliteDSN(config), sqliteInit
}

// SqliteDSN takes a database configuration and returns the corresponding
// SQLite DSN necessary to connect to the database.
func SqliteDSN(config *conf.DatabaseConfig) string {
	var user, pass string
	if config.User != "" {
		user = fmt.Sprintf("&_auth_user=%s", config.User)
	}

	if config.Password != "" {
		pass = fmt.Sprintf("&_auth_pass=%s", config.Password)
	}

	return fmt.Sprintf("file:%s?mode=rwc&_busy_timeout=10000%s%s",
		config.Address, user, pass)
}
