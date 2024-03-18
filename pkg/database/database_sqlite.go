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
	SupportedRBMS[SQLite] = sqliteinfo
}

func sqliteinfo() *DBInfo {
	return &DBInfo{
		Driver: migrations.SqliteDriver,
		DSN:    migrations.SqliteDSN(),
	}
}
