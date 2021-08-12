package migrations

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/migration"
)

type dbInfo struct {
	driver  string
	makeDSN func(*conf.DatabaseConfig) string
}

var rdbms = map[string]dbInfo{
	migration.SQLite: {
		driver:  database.SqliteDriver,
		makeDSN: database.SqliteDSN,
	},
	migration.PostgreSQL: {
		driver:  database.PostgresDriver,
		makeDSN: database.PostgresDSN,
	},
	migration.MySQL: {
		driver:  database.MysqlDriver,
		makeDSN: database.MysqlDSN,
	},
}
