package migrations

import (
	"database/sql"
	"errors"
	"fmt"
	"io"

	"golang.org/x/mod/semver"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/migration"
)

const windowsRuntime = "windows"

var errInvalidVersion = errors.New("invalid database version")

func getTarget(version string) (index int, err error) {
	if version == "latest" {
		return len(Migrations), nil
	}

	if !semver.IsValid("v" + version) {
		return -1, fmt.Errorf("bad version %q: %w", version, errInvalidVersion)
	}

	for i, m := range Migrations {
		if m.VersionTag == version {
			return i, nil
		}
	}

	return -1, fmt.Errorf("bad version %q: %w", version, errInvalidVersion)
}

func getCurrent(db *sql.DB, dialect string) (index int, err error) {
	ok, err := checkVersionTableExist(db, dialect)
	if err != nil {
		return -1, fmt.Errorf("failed to check the database version table: %w", err)
	}

	if !ok { // if version table does not exist, start migrations from the beginning
		return 0, nil
	}

	row := db.QueryRow("SELECT current FROM version")

	var current string
	if err := row.Scan(&current); err != nil {
		return -1, fmt.Errorf("cannot scan database results: %w", err)
	}

	for i, m := range Migrations {
		if m.VersionTag == current {
			return i, nil
		}
	}

	return -1, fmt.Errorf("the current database version (%s) is unknown: %w", current, errInvalidVersion)
}

func doMigration(db *sql.DB, version, dialect string, out io.Writer) error {
	current, err := getCurrent(db, dialect)
	if err != nil {
		return err
	}

	target, err := getTarget(version)
	if err != nil {
		return err
	}

	if target == current {
		return nil // nothing to do
	}

	engine, err := migration.NewEngine(db, dialect, out)
	if err != nil {
		return fmt.Errorf("cannot initialize migration engine: %w", err)
	}

	if target > current {
		if err := engine.Upgrade(Migrations[current+1 : target+1]); err != nil {
			return fmt.Errorf("cannot upgrade database: %w", err)
		}

		return nil
	}

	if err := engine.Downgrade(Migrations[target+1 : current+1]); err != nil {
		return fmt.Errorf("cannot downgrade database: %w", err)
	}

	return nil
}

// Execute migrates the database given in the configuration from its current
// version to the one given as parameter.
func Execute(config *conf.DatabaseConfig, version string, out io.Writer) error {
	dbInfo, ok := rdbms[config.Type]
	if !ok {
		return fmt.Errorf("unknown RDBMS %s: %w", config.Type, errUnsuportedDB)
	}

	db, err := sql.Open(dbInfo.driver, dbInfo.makeDSN(config))
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	defer func() { _ = db.Close() }() //nolint:errcheck // cannot handle the error

	return doMigration(db, version, config.Type, out)
}
