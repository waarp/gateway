package migrations

import (
	"database/sql"
	"errors"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/version"
)

const VersionNone = "none"

//nolint:gochecknoglobals,mnd //a global var with magic numbers is required here
var VersionsMap = map[string]int{
	"0.4.0":  0,
	"0.4.1":  0,
	"0.4.2":  1,
	"0.4.3":  1,
	"0.4.4":  1,
	"0.5.0":  14,
	"0.5.1":  14,
	"0.5.2":  15,
	"0.6.0":  16,
	"0.6.1":  16,
	"0.6.2":  16,
	"0.7.0":  31,
	"0.7.1":  31,
	"0.7.2":  31,
	"0.7.3":  31,
	"0.7.4":  31,
	"0.7.5":  32,
	"0.8.0":  36,
	"0.8.1":  36,
	"0.8.2":  36,
	"0.9.0":  51,
	"0.9.1":  51,
	"0.10.0": 53,
	"0.10.1": 53,

	VersionNone: -1,
	version.Num: len(Migrations) - 1,
}

func setDBVersion(to string) func(Actions) error {
	return func(db Actions) error {
		if err := db.Exec("UPDATE version SET current=?", to); err != nil {
			return fmt.Errorf("cannot set data model version: %w", err)
		}

		return nil
	}
}

func checkVersionTableExist(db *sql.DB, dialect string) (bool, error) {
	var row *sql.Row

	switch dialect {
	case SQLite:
		row = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='version'")
	case PostgreSQL:
		row = db.QueryRow("SELECT tablename FROM pg_tables WHERE tablename='version'")
	case MySQL:
		row = db.QueryRow("SHOW TABLES LIKE 'version'")
	default:
		return false, fmt.Errorf("%w: %q", ErrUnknownDialect, dialect)
	}

	var name any
	if err := row.Scan(&name); errors.Is(err, sql.ErrNoRows) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to check the version table: %w", err)
	}

	return true, nil
}
