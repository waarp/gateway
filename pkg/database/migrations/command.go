package migrations

import (
	"database/sql"
	"errors"
	"fmt"
	"io"

	"golang.org/x/mod/semver"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/migration"
	vers "code.waarp.fr/apps/gateway/gateway/pkg/version"
)

const windowsRuntime = "windows"

var errInvalidVersion = errors.New("invalid database version")

func getTarget(version string) (index int, err error) {
	switch version {
	case "latest-testing":
		Migrations = append(Migrations, migration.Migration{Script: bumpVersion{to: vers.Num}})

		return len(Migrations), nil
	case "latest":
		return len(Migrations), nil
	default:
	}

	if !semver.IsValid("v" + version) {
		return -1, fmt.Errorf("bad target version %q: %w", version, errInvalidVersion)
	}

	for i, m := range Migrations {
		if m.VersionTag == version {
			return i + 1, nil
		}
	}

	return -1, fmt.Errorf("bad target version %q: %w", version, errInvalidVersion)
}

func getStart(db *sql.DB, dialect string, from int) (index int, err error) {
	if from > 0 && from < len(Migrations) {
		return from, nil
	}

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
			return i + 1, nil
		}
	}

	return -1, fmt.Errorf("the current database version (%s) is unknown: %w", current, errInvalidVersion)
}

func doMigration(db *sql.DB, version, dialect string, from int, out io.Writer) error {
	start, err := getStart(db, dialect, from)
	if err != nil {
		return err
	}

	target, err := getTarget(version)
	if err != nil {
		return err
	}

	if target == start {
		return nil // nothing to do
	}

	engine, err := migration.NewEngine(db, dialect, out)
	if err != nil {
		return fmt.Errorf("cannot initialize migration engine: %w", err)
	}

	if target > start {
		if err := engine.Upgrade(Migrations[start:target]); err != nil {
			return fmt.Errorf("cannot upgrade database: %w", err)
		}

		return nil
	}

	if err := engine.Downgrade(Migrations[target:start]); err != nil {
		return fmt.Errorf("cannot downgrade database: %w", err)
	}

	return nil
}

// Execute migrates the database given in the configuration from its current
// version to the one given as parameter.
func Execute(config *conf.DatabaseConfig, version string, from int, out io.Writer) error {
	dbInfo, ok := rdbms[config.Type]
	if !ok {
		return fmt.Errorf("unknown RDBMS %s: %w", config.Type, errUnsuportedDB)
	}

	db, err := sql.Open(dbInfo.driver, dbInfo.makeDSN())
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	defer func() { _ = db.Close() }() //nolint:errcheck // cannot handle the error

	return doMigration(db, version, config.Type, from, out)
}
