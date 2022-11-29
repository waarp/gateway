package database

import (
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

func sqliteinfo() *dbInfo {
	return &dbInfo{
		driver: migrations.SqliteDriver,
		dsn:    migrations.SqliteDSN(),
	}
}
