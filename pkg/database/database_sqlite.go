package database

import (
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

func sqliteInit(*xorm.Engine) error { return nil }

func sqliteinfo() (string, string, func(*xorm.Engine) error) {
	return migrations.SqliteDriver, migrations.SqliteDSN(), sqliteInit
}
