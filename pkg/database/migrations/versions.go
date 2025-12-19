package migrations

import (
	"database/sql"
	"errors"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/version"
)

const VersionNone = "none"

// VersionsMap pairs each Gateway version with its corresponding index in the
// migrations list. A rule of thumb is that when adding a new version, the
// number associated with it should be equal to the index of the last element in
// the Migrations list (not then list's length).
//
//nolint:gochecknoglobals,mnd //a global var with magic numbers is required here
var VersionsMap = map[string]int{
	"0.4.0":   0,
	"0.4.1":   0,
	"0.4.2":   1,
	"0.4.3":   1,
	"0.4.4":   1,
	"0.5.0":   14,
	"0.5.1":   14,
	"0.5.2":   15,
	"0.6.0":   16,
	"0.6.1":   16,
	"0.6.2":   16,
	"0.7.0":   31,
	"0.7.1":   31,
	"0.7.2":   31,
	"0.7.3":   31,
	"0.7.4":   31,
	"0.7.5":   32,
	"0.8.0":   36,
	"0.8.1":   36,
	"0.8.2":   36,
	"0.9.0":   51,
	"0.9.1":   51,
	"0.10.0":  53,
	"0.10.1":  53,
	"0.11.0":  55,
	"0.11.1":  55,
	"0.11.2":  55,
	"0.11.3":  55,
	"0.11.4":  55,
	"0.11.5":  55,
	"0.11.6":  55,
	"0.12.0":  57,
	"0.12.1":  58,
	"0.12.2":  58,
	"0.12.3":  58,
	"0.12.4":  58,
	"0.12.5":  58,
	"0.12.6":  58,
	"0.12.8":  58,
	"0.12.9":  58,
	"0.12.10": 58,
	"0.12.11": 58,
	"0.13.0":  62,
	"0.13.1":  62,
	"0.13.2":  62,
	"0.13.3":  62,
	"0.14.0":  63,

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
