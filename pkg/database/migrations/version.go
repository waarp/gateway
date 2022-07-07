package migrations

import (
	"database/sql"
	"errors"
	"fmt"

	"code.waarp.fr/lib/log"
	"code.waarp.fr/lib/migration"
	"github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/version"
)

var errUnsuportedDB = errors.New("unsupported database")

// bumpVersion is a migration scripts which bump the database version from a
// given version to another.
type bumpVersion struct{ from, to string }

func (b bumpVersion) Up(db migration.Actions) error {
	if err := db.Exec("UPDATE version SET current=?", b.to); err != nil {
		return fmt.Errorf("cannot set data model version: %w", err)
	}

	return nil
}

func (b bumpVersion) Down(db migration.Actions) error {
	if err := db.Exec("UPDATE version SET current=?", b.from); err != nil {
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

// BumpToCurrent returns a migration script to bump the database version to the
// current program version. Use only for testing.
func BumpToCurrent(c convey.C, db *sql.DB, logger *log.Logger, dialect string) {
	engine, err := migration.NewEngine(db, dialect, logger, nil)
	c.So(err, convey.ShouldBeNil)

	toApply := makeMigration(Migrations)
	toApply = append(toApply, migration.Script{
		Description: fmt.Sprintf("Bump database version to %s", version.Num),
		Up:          bumpVersion{to: version.Num}.Up,
		Down:        nil,
	})

	c.So(engine.Upgrade(toApply), convey.ShouldBeNil)
}
