package database

import (
	"time"

	"xorm.io/xorm"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
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

	return nil
}

func sqliteinfo(config *conf.DatabaseConfig) (string, string, func(*xorm.Engine) error) {
	return migrations.SqliteDriver, migrations.SqliteDSN(config), sqliteInit
}
