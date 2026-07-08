package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/migrations"
)

const (
	// PostgreSQL is the configuration option for using the PostgreSQL RDBMS.
	PostgreSQL = "postgresql"

	postgresDialector = "postgres" // from postgres.Dialector.Name()
)

//nolint:gochecknoinits // init is used by design
func init() {
	SupportedRBMS[PostgreSQL] = postgresinfo
}

func postgresinfo(config *conf.DatabaseConfig) (gorm.Dialector, error) {
	return postgres.New(
		postgres.Config{
			DriverName: migrations.PostgresDriver,
			DSN:        migrations.PostgresDSN(config),
		},
	), nil
}
