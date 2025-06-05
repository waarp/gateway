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
	inits []Initialiser

	// BcryptRounds defines the number of rounds taken by bcrypt to hash passwords
	// in the database.
	BcryptRounds = 12
)

// AddInit adds the given model to the pool of table initializations.
func AddInit(t Initialiser) { inits = append(inits, t) }

// Initialiser is an interface which models can optionally implement in order to
// set default values after the table is created when the application is launched
// for the first time.
type Initialiser interface {
	Table
	// Init initializes the table.
	Init(db Access) error
}

type Executor struct {
	Dialect string
	Logger  *log.Logger
	ses     *xorm.Session
}

func (e *Executor) Exec(query string, args ...any) (sql.Result, error) {
	elems := append([]any{query}, args...)

	res, err := e.ses.Exec(elems...)
	if err != nil {
		return nil, fmt.Errorf("command failed: %w", err)
	}

	return res, nil
}

func (e *Executor) Query(query string, args ...any) ([]map[string]any, error) {
	elems := append([]any{query}, args...)

	res, err := e.ses.QueryInterface(elems...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	return res, nil
}

// initDatabase initializes the database and then updates it to the latest version.
func initDatabase(db *Standalone) error {
	sqlDB := db.engine.DB().DB
	dialect := conf.GlobalConfig.Database.Type
	logger := db.logger

	dbExist, existErr := db.engine.IsTableExist(&version{})
	if existErr != nil {
		logger.Criticalf("Failed to probe the database table list: %v", existErr)

		return fmt.Errorf("failed to probe the database table list: %w", existErr)
	}

	if !dbExist {
		if err2 := migrations.DoMigration(sqlDB, logger, vers.Num, dialect, nil); err2 != nil {
			logger.Criticalf("Database initialization failed: %v", err2)

			return fmt.Errorf("database initialization failed: %w", err2)
		}
	}

	if err := db.Transaction(initTables); err != nil {
		if !dbExist {
			if err2 := migrations.DoMigration(sqlDB, logger, migrations.VersionNone,
				dialect, nil); err2 != nil {
				logger.Warningf("Failed to restore the pristine database: %v", err2)
			}
		}

		return fmt.Errorf("failed to initialize tables: %w", err)
	}

	return nil
}

func initTables(ses *Session) error {
	for _, init := range inits {
		if err := init.Init(ses); err != nil {
			ses.logger.Errorf("Failed to initialize table %q: %v", init.TableName(), err)

			return fmt.Errorf("failed to initialize table %q: %w", init.TableName(), err)
		}
	}

	return nil
}
