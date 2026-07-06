package database

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/migrations"
)

const (
	// SQLite is the configuration option for using the SQLite RDBMS.
	SQLite       = "sqlite"
	SQLiteDriver = migrations.SqliteDriver

	sqliteDialector = "sqlite" // from sqlite.Dialector.Name()
)

//nolint:gochecknoinits // init is used by design
func init() {
	SupportedRBMS[SQLite] = sqliteinfo
}

func sqliteinfo(config *conf.DatabaseConfig) (gorm.Dialector, error) {
	return sqlite.New(
		sqlite.Config{
			DriverName: migrations.SqliteDriver,
			DSN:        migrations.SqliteDSN(config),
		},
	), nil
}
