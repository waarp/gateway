package database

import (
	"xorm.io/xorm"

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

func mysqlInit(*xorm.Engine) error { return nil }

func mysqlinfo() (string, string, func(*xorm.Engine) error) {
	return migrations.MysqlDriver, migrations.MysqlDSN(), mysqlInit
}
