package database

import (
	"fmt"
	"time"

	"xorm.io/xorm"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/migrations"
)

const (
	// SQLite is the configuration option for using the SQLite RDBMS.
	SQLite = "sqlite"
)

//nolint:gochecknoinits // init is used by design
func init() {
	supportedRBMS[SQLite] = sqliteinfo
}

func sqliteInit(db *xorm.Engine) error {
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.DatabaseTZ = time.UTC

	if _, err := db.Exec("PRAGMA busy_timeout = 10000"); err != nil {
		return fmt.Errorf("failed to set busy timeout: %w", err)
	}

	return nil
}

func sqliteinfo() (string, string, func(*xorm.Engine) error) {
	return migrations.SqliteDriver, migrations.SqliteDSN(), sqliteInit
}
