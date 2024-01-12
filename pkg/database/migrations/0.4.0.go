package migrations

import (
	"fmt"
	"strings"

	"code.waarp.fr/lib/migration"
)

func ver0_4_0InitDatabaseUp(db Actions) error {
	var initScript string

	switch dial := db.GetDialect(); dial {
	case SQLite:
		initScript = strings.TrimSpace(SqliteCreationScript)
	case PostgreSQL:
		initScript = strings.TrimSpace(PostgresCreationScript)
	case MySQL:
		initScript = strings.TrimSpace(MysqlCreationScript)
	default:
		return errUnknownDialect(dial)
	}

	for _, statement := range strings.Split(initScript, ";\n") {
		if err := db.Exec(statement); err != nil {
			return fmt.Errorf("failed to init the database: %w", err)
		}
	}

	return nil
}

func ver0_4_0InitDatabaseDown(db migration.Actions) error {
	for _, table := range initTableList() {
		if err := db.DropTable(table); err != nil {
			return fmt.Errorf("failed to drop table %q: %w", table, err)
		}
	}

	return nil
}
