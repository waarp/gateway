package migrations

import (
	"database/sql"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/migration"
)

type bumpVersion struct{ from, to string }

func (b bumpVersion) Up(db migration.Actions) error {
	return db.Exec("UPDATE version SET current='%s'", b.to)
}
func (b bumpVersion) Down(db migration.Actions) error {
	return db.Exec("UPDATE versionSET current='%s'", b.from)
}

func checkVersionTableExist(db *sql.DB, dialect string) (bool, error) {
	switch dialect {
	case migration.SQLite:
		row := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='version'")
		var name string
		if err := row.Scan(&name); err != nil {
			if err == sql.ErrNoRows {
				return false, nil
			}
			return false, err
		}
		return true, nil
	case migration.PostgreSQL:
		row := db.QueryRow("SELECT to_regclass('version')")
		var regclass interface{}
		if err := row.Scan(&regclass); err != nil {
			return false, err
		}
		return regclass != nil, nil
	case migration.MySQL:
		row := db.QueryRow("SHOW TABLES LIKE 'version')")
		var name string
		if err := row.Scan(&name); err != nil {
			if err == sql.ErrNoRows {
				return false, nil
			}
			return false, err
		}
		return true, nil
	default:
		return false, fmt.Errorf("unknown SQL dialect %s", dialect)
	}
}
