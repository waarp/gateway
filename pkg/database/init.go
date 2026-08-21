package database

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/migrations"
	vers "code.waarp.fr/apps/gateway/gateway/pkg/version"
)

type Initializer struct {
	Desc string
	Func func(Access) error
}

//nolint:gochecknoglobals // global var is used by design
var (
	inits []Initializer

	// BcryptRounds defines the number of rounds taken by bcrypt to hash passwords
	// in the database.
	BcryptRounds = 12
)

// AddInit adds the database init function to the list of initializers.
func AddInit(init Initializer) { inits = append(inits, init) }

// initDatabase initializes the database and then updates it to the latest version.
func (db *DB) initDatabase() error {
	sqlDB, sqlErr := db.engine.DB()
	if sqlErr != nil {
		return fmt.Errorf("failed to retrieve db connection: %w", sqlErr)
	}

	dialect := db.Config.Database.Type
	logger := db.Logger
	dbExist := db.engine.Migrator().HasTable(&version{})

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
		if err := init.Func(ses); err != nil {
			ses.getLogger().Errorf("Failed to run initializer %q: %v", init.Desc, err)

			return fmt.Errorf("failed to run initializer %q: %w", init.Desc, err)
		}
	}

	return nil
}
