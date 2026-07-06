// Package database contains the module for accessing the gateway's database.
package database

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gorm.io/gorm"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	aesKeySize = 32

	ServiceName = "Database"
)

var errUnsupportedDB = errors.New("unsupported database")

// DB is the database service. It encapsulates a data connection and implements
// Accessor.
type DB struct {
	Logger *log.Logger
	Config *conf.ServerConfig
	AEAD   cipher.AEAD

	engine *gorm.DB
	state  utils.State
}

func NewDB(config *conf.ServerConfig) *DB {
	return &DB{Config: config}
}

func (db *DB) Name() string { return ServiceName }

func (db *DB) ChangeAEAD(newAEAD cipher.AEAD) {
	db.AEAD = newAEAD

	if db.engine != nil {
		registerAEAD(db.engine, newAEAD)
	}
}

func (db *DB) loadAESKey() error {
	if db.AEAD != nil {
		return nil
	}

	filename := db.Config.Database.AESPassphrase
	if _, statErr := os.Stat(filepath.Clean(filename)); os.IsNotExist(statErr) {
		db.Logger.Infof("Creating AES passphrase file at %q", filename)

		key := make([]byte, aesKeySize)

		if _, err := rand.Read(key); err != nil {
			return fmt.Errorf("cannot generate AES key: %w", err)
		}

		if err := os.WriteFile(filepath.Clean(filename), key, 0o600); err != nil {
			return fmt.Errorf("cannot write AES key to file %q: %w", filename, err)
		}
	}

	var gcmErr error
	db.AEAD, gcmErr = NewAEAD(filename)

	return gcmErr
}

func NewAEAD(filename string) (cipher.AEAD, error) {
	key, err := os.ReadFile(filepath.Clean(filename))
	if err != nil {
		return nil, fmt.Errorf("cannot read AES key from file %q: %w", filename, err)
	}

	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize AES key: %w", err)
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize AES key: %w", err)
	}

	return gcm, nil
}

// makeDialector creates and returns the dataSourceName string necessary
// to open a connection to the database, along with the driver and an optional
// initialisation function. The DSN varies depending on the options given
// in the database configuration.
func (db *DB) makeDialector() (gorm.Dialector, error) {
	rdbms := db.Config.Database.Type

	makeConnInfo, ok := SupportedRBMS[rdbms]
	if !ok {
		return nil, fmt.Errorf("unknown database type '%s': %w", rdbms, errUnsupportedDB)
	}

	return makeConnInfo(&db.Config.Database)
}

type connMaker func(config *conf.DatabaseConfig) (gorm.Dialector, error)

//nolint:gochecknoglobals // global var is used by design
var SupportedRBMS = map[string]connMaker{}

func (db *DB) initEngine() error {
	dialector, err := db.makeDialector()
	if err != nil {
		db.Logger.Criticalf("Database configuration invalid: %v", err)

		return err
	}

	db.engine, err = gorm.Open(dialector, &gorm.Config{
		Logger:            &gormLogger{Logger: db.Logger},
		AllowGlobalUpdate: true,
	})
	if err != nil {
		db.Logger.Criticalf("Failed to open database: %v", err)

		return fmt.Errorf("cannot initialize database access: %w", err)
	}

	registerAEAD(db.engine, db.AEAD)

	return nil
}

// Start launches the database service using the configuration given in the
// Environment field. If the configuration is invalid, or if the database
// cannot be reached, an error is returned.
func (db *DB) Start() error {
	if db.state.IsRunning() {
		return nil
	}

	if err := db.start(true); err != nil {
		db.state.Set(utils.StateError, err.Error())

		return err
	}

	db.state.Set(utils.StateRunning, "")

	return nil
}

func (db *DB) start(withInit bool) (retErr error) {
	if db.Logger == nil {
		db.Logger = logging.NewLogger(ServiceName)
	}
	db.Logger.Info("Starting database service...")

	if err := db.loadAESKey(); err != nil {
		db.Logger.Criticalf("Failed to load AES key: %v", err)

		return err
	}

	if err := db.initEngine(); err != nil {
		return err
	}

	defer func() {
		if retErr != nil {
			_ = db.close()
		}
	}()

	if err := db.checkVersion(); err != nil {
		return err
	}

	if withInit {
		if err := db.initDatabase(); err != nil {
			return err
		}
	}

	db.Logger.Info("Startup successful")

	return nil
}

// Stop shuts down the database service. If an error occurred during the shutdown,
// an error is returned.
// If the service is not running, this function does nothing.
func (db *DB) Stop(context.Context) error {
	if !db.state.IsRunning() {
		return utils.ErrNotRunning
	}

	if err := db.stop(); err != nil {
		db.state.Set(utils.StateError, err.Error())

		return err
	}

	db.state.Set(utils.StateOffline, "")

	return nil
}

func (db *DB) stop() error {
	defer func() { db.engine = nil }()

	db.Logger.Info("Shutting down...")

	if err := db.close(); err != nil {
		db.Logger.Infof("Error while closing the database: %v", err)

		return fmt.Errorf("an error occurred while closing the database: %w", err)
	}

	db.Logger.Info("Shutdown complete")
	db.engine = nil

	return nil
}

// State returns the state of the database service.
func (db *DB) State() (utils.StateCode, string) {
	return db.state.Get()
}

//nolint:wrapcheck //no need to wrap errors here
func (db *DB) close() error {
	if db.engine == nil {
		return nil
	}

	sqlDB, err := db.engine.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}
