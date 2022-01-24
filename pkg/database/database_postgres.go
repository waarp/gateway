package database

import (
	"xorm.io/xorm"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
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

func postgresinfo(config *conf.DatabaseConfig) (string, string, func(*xorm.Engine) error) {
	return migrations.PostgresDriver, migrations.PostgresDSN(config), func(*xorm.Engine) error {
		return nil
	}
}
