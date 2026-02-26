package migrations

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"runtime"

	"code.waarp.fr/lib/migration"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
)

func isWindowsRuntime() bool {
	return runtime.GOOS == "windows"
}

var errInvalidVersion = errors.New("invalid database version")

func getTargetIndex(target string) (int, error) {
	index, ok := VersionsMap[target]
	if !ok {
		return -1, fmt.Errorf("the target database version (%s) is unknown: %w",
			target, errInvalidVersion)
	}

	return index, nil
}

func getCurrentIndex(db *sql.DB, dialect string) (int, error) {
	ok, err := checkVersionTableExist(db, dialect)
	if err != nil {
		return 0, fmt.Errorf("failed to check the database version table: %w", err)
	}

	if !ok { // if version table does not exist, start migrations from the beginning
		return -1, nil
	}

	row := db.QueryRow("SELECT current FROM version")

	var current string
	if scanErr := row.Scan(&current); scanErr != nil {
		return 0, fmt.Errorf("cannot scan database results: %w", scanErr)
	}

	index, ok := VersionsMap[current]
	if !ok {
		return 0, fmt.Errorf("the current database version (%s) is unknown: %w",
			current, errInvalidVersion)
	}

	return index, nil
}

func DoMigration(db *sql.DB, logger *log.Logger, targetVersion, dialect string, out io.Writer,
) error {
	start, err := getCurrentIndex(db, dialect)
	if err != nil {
		return err
	}

	target, err := getTargetIndex(targetVersion)
	if err != nil {
		return err
	}

	versionBump := []migration.Script{{
		Description: fmt.Sprintf("Bump database version to %s", targetVersion),
		Up:          setDBVersion(targetVersion),
		Down:        setDBVersion(targetVersion),
	}}

	engine, err := migration.NewEngine(db, dialect, logger, out)
	if err != nil {
		return fmt.Errorf("cannot initialize migration engine: %w", err)
	}

	if target >= start {
		toApply := make([]migration.Script, target-start, target-start+1)
		copy(toApply, Migrations[start+1:target+1])
		toApply = append(toApply, versionBump...)

		if upErr := engine.Upgrade(toApply); upErr != nil {
			return fmt.Errorf("cannot upgrade database: %w", upErr)
		}

		return nil
	}

	toApply := make([]migration.Script, start-target, start-target+1)
	copy(toApply, Migrations[target+1:start+1])
	toApply = append(versionBump, toApply...)

	if downErr := engine.Downgrade(toApply); downErr != nil {
		return fmt.Errorf("cannot downgrade database: %w", downErr)
	}

	return nil
}

// Execute migrates the database given in the configuration from its current
// version to the one given as parameter.
func Execute(config *conf.DatabaseConfig, logger *log.Logger, targetVersion string,
	out io.Writer,
) error {
	dbInfo, ok := rdbms[config.Type]
	if !ok {
		return errUnknownDialect(config.Type)
	}

	db, err := sql.Open(dbInfo.driver, dbInfo.makeDSN())
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	db.SetMaxOpenConns(1)

	defer func() { _ = db.Close() }() //nolint:errcheck // cannot handle the error

	return DoMigration(db, logger, targetVersion, config.Type, out)
}
