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
	db.DatabaseTZ = time.UTC

	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return fmt.Errorf("failed to activate SQLite foreign keys: %w", err)
	}

	return nil
}

func sqliteinfo() (string, string, func(*xorm.Engine) error) {
	return migrations.SqliteDriver, migrations.SqliteDSN(), sqliteInit
}
