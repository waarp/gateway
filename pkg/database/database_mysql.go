package database

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/migrations"
)

const (
	// MySQL is the configuration option for using the MySQL RDBMS.
	MySQL = "mysql"

	mysqlDialector = mysql.DefaultDriverName
)

//nolint:gochecknoinits // init is used by design
func init() {
	SupportedRBMS[MySQL] = mysqlinfo
}

func mysqlinfo(config *conf.DatabaseConfig) (gorm.Dialector, error) {
	return mysql.New(
		mysql.Config{
			DriverName: migrations.MysqlDriver,
			DSN:        migrations.MysqlDSN(config),
		},
	), nil
}
