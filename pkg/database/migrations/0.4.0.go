package migrations

import (
	"fmt"
	"strings"

	"code.waarp.fr/lib/migration"
)

type ver0_4_0InitDatabase struct{}

func (ver0_4_0InitDatabase) Up(db migration.Actions) error {
	var initScript string

	switch db.GetDialect() {
	case SQLite:
		initScript = strings.TrimSpace(SqliteCreationScript)
	case PostgreSQL:
		initScript = strings.TrimSpace(PostgresCreationScript)
	case MySQL:
		initScript = strings.TrimSpace(MysqlCreationScript)
	default:
		return fmt.Errorf("unknown dialect engine %T: %w", db, errUnsuportedDB)
	}

	for _, statement := range strings.Split(initScript, ";\n") {
		if err := db.Exec(statement); err != nil {
			return fmt.Errorf("failed to init the database: %w", err)
		}
	}

	return nil
}

func (v ver0_4_0InitDatabase) Down(db migration.Actions) error {
	for _, table := range initTableList() {
		if err := db.DropTable(table); err != nil {
			return fmt.Errorf("failed to drop table %q: %w", table, err)
		}
	}

	return nil
}
