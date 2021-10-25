package migrations

import (
	"database/sql"
	"errors"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/tk/migration"
)

var errUnsuportedDB = errors.New("unsupported database")

// bumpVersion is a migration scripts which bump the database version from a
// given version to another.
type bumpVersion struct{ from, to string }

func (b bumpVersion) Up(db migration.Actions) error {
	if err := db.Exec("UPDATE version SET current='%s'", b.to); err != nil {
		return fmt.Errorf("cannot set data model version: %w", err)
	}

	return nil
}

func (b bumpVersion) Down(db migration.Actions) error {
	if err := db.Exec("UPDATE version SET current='%s'", b.from); err != nil {
		return fmt.Errorf("cannot set data model version: %w", err)
	}

	return nil
}

func checkVersionTableExist(db *sql.DB, dialect string) (bool, error) {
	switch dialect {
	case migration.SQLite:
		row := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='version'")

		var name string
		if err := row.Scan(&name); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return false, nil
			}

			return false, fmt.Errorf("cannot scan results from the database: %w", err)
		}

		return true, nil
	case migration.PostgreSQL:
		row := db.QueryRow("SELECT to_regclass('version')")

		var regclass interface{}
		if err := row.Scan(&regclass); err != nil {
			return false, fmt.Errorf("cannot scan results from the database: %w", err)
		}

		return regclass != nil, nil
	case migration.MySQL:
		row := db.QueryRow("SHOW TABLES LIKE 'version')")

		var name string
		if err := row.Scan(&name); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return false, nil
			}

			return false, fmt.Errorf("cannot scan results from database: %w", err)
		}

		return true, nil
	default:
		return false, fmt.Errorf("unknown SQL dialect %s: %w", dialect, errUnsuportedDB)
	}
}
