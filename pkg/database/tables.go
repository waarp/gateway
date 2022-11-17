package database

import (
	"database/sql"
	"fmt"

	"code.waarp.fr/lib/log"
	"xorm.io/xorm"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/migrations"
	vers "code.waarp.fr/apps/gateway/gateway/pkg/version"
)

//nolint:gochecknoglobals // global var is used by design
var (
	// Tables lists the schema of all database tables.
	tables []Table

	// BcryptRounds defines the number of rounds taken by bcrypt to hash passwords
	// in the database.
	BcryptRounds = 12
)

// AddTable adds the given model to the pool of database tables.
func AddTable(t Table) {
	tables = append(tables, t)
}

// initialiser is an interface which models can optionally implement in order to
// set default values after the table is created when the application is launched
// for the first time.
type initialiser interface {
	Init(Access) Error
}

type Executor struct {
	Dialect string
	Logger  *log.Logger
	ses     *xorm.Session
}

func (e *Executor) Exec(query string, args ...interface{}) (sql.Result, error) {
	elems := append([]interface{}{query}, args...)

	if res, err := e.ses.Exec(elems...); err != nil {
		return nil, fmt.Errorf("command failed: %w", err)
	} else {
		return res, nil
	}
}

func (e *Executor) Query(query string, args ...interface{}) ([]map[string]interface{}, error) {
	elems := append([]interface{}{query}, args...)

	if res, err := e.ses.QueryInterface(elems...); err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	} else {
		return res, nil
	}
}

// initDatabase initializes the database and then updates it to the latest version.
func initDatabase(db *Standalone) error {
	sqlDB := db.engine.DB().DB
	dialect := conf.GlobalConfig.Database.Type
	logger := db.logger

	dbExist, existErr := db.engine.IsTableExist(&version{})
	if existErr != nil {
		logger.Critical("Failed to probe the database table list: %v", existErr)

		return fmt.Errorf("failed to probe the database table list: %w", existErr)
	}

	if !dbExist {
		if err2 := migrations.DoMigration(sqlDB, logger, vers.Num, dialect, nil); err2 != nil {
			logger.Critical("Database initialization failed: %v", err2)

			return fmt.Errorf("database initialization failed: %w", err2)
		}
	}

	if err := db.Transaction(initTables); err != nil {
		if !dbExist {
			if err2 := migrations.DoMigration(sqlDB, logger, migrations.VersionNone,
				dialect, nil); err2 != nil {
				logger.Warning("Failed to restore the pristine database: %v", err2)
			}
		}

		return fmt.Errorf("failed to initialize tables: %w", err)
	}

	return nil
}

func initTables(ses *Session) Error {
	for _, tbl := range tables {
		if init, ok := tbl.(initialiser); ok {
			if err := init.Init(ses); err != nil {
				ses.logger.Error("failed to initialize table %q: %v", tbl.TableName(), err)

				return err
			}
		}
	}

	return nil
}
