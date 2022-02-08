package database

import (
	"time"

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

func mysqlInit(db *xorm.Engine) error {
	db.DatabaseTZ = time.UTC

	return nil
}

func mysqlinfo() (string, string, func(*xorm.Engine) error) {
	return migrations.MysqlDriver, migrations.MysqlDSN(), mysqlInit
}
