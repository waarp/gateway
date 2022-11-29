package database

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database/migrations"
)

const (
	// MySQL is the configuration option for using the MySQL RDBMS.
	MySQL = "mysql"
)

//nolint:gochecknoinits // init is used by design
func init() {
	supportedRBMS[MySQL] = mysqlinfo
}

func mysqlinfo() *dbInfo {
	return &dbInfo{
		driver: migrations.MysqlDriver,
		dsn:    migrations.MysqlDSN(),
	}
}
