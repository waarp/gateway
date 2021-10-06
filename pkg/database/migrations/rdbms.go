package migrations

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	migration2 "code.waarp.fr/apps/gateway/gateway/pkg/tk/migration"
)

type dbInfo struct {
	driver  string
	makeDSN func(*conf.DatabaseConfig) string
}

//nolint:gochecknoglobals // global var is used by design
var rdbms = map[string]dbInfo{
	migration2.SQLite: {
		driver:  database.SqliteDriver,
		makeDSN: database.SqliteDSN,
	},
	migration2.PostgreSQL: {
		driver:  database.PostgresDriver,
		makeDSN: database.PostgresDSN,
	},
	migration2.MySQL: {
		driver:  database.MysqlDriver,
		makeDSN: database.MysqlDSN,
	},
}
