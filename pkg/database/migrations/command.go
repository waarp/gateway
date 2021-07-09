package migrations

import (
	"database/sql"
	"fmt"
	"io"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/migration"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"golang.org/x/mod/semver"
)

func getTarget(version string) (index int, err error) {
	invalid := fmt.Errorf("%s is not a valid database version", version)
	if version == "latest" {
		return len(Migrations), nil
	}

	if !semver.IsValid("v" + version) {
		return -1, invalid
	}

	for i, m := range Migrations {
		if m.Version == version {
			return i, nil
		}
	}
	return -1, invalid
}

func getCurrent(db *sql.DB, dialect string) (index int, err error) {
	ok, err := checkVersionTableExist(db, dialect)
	if err != nil {
		return -1, fmt.Errorf("failed to check the database version table: %s", err)
	}
	if !ok { // if version table does not exist, start migrations from the beginning
		return 0, nil
	}

	row := db.QueryRow("SELECT current FROM version")

	var current string
	if err := row.Scan(&current); err != nil {
		return -1, err
	}

	for i, m := range Migrations {
		if m.Version == current {
			return i, nil
		}
	}
	return -1, fmt.Errorf("the current database version (%s) is unknown", current)
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
		return nil //nothing to do
	}

	engine, err := migration.NewEngine(db, dialect, out)
	if err != nil {
		return err
	}

	if target > current {
		return engine.Upgrade(Migrations[current:target])
	}
	return engine.Downgrade(Migrations[target:current])
}

// Execute migrates the database given in the configuration from its current
// version to the one given as parameter.
func Execute(conf *conf.DatabaseConfig, version string, out io.Writer) error {
	dbInfo, ok := rdbms[conf.Type]
	if !ok {
		return fmt.Errorf("unknown RDBMS %s", conf.Type)
	}

	db, err := sql.Open(dbInfo.driver, dbInfo.makeDSN(conf))
	if err != nil {
		return fmt.Errorf("failed to connect to database: %s", err)
	}
	defer func() { _ = db.Close() }()

	return doMigration(db, version, conf.Type, out)
}
