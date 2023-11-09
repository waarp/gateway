package database

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database/migrations"
)

const (
	// PostgreSQL is the configuration option for using the PostgreSQL RDBMS.
	PostgreSQL = "postgresql"
)

//nolint:gochecknoinits // init is used by design
func init() {
	supportedRBMS[PostgreSQL] = postgresinfo
}

func postgresinfo() *DBInfo {
	return &DBInfo{
		Driver: migrations.PostgresDriver,
		DSN:    migrations.PostgresDSN(),
	}
}
