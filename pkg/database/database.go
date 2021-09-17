// Package database contains the module for accessing the gateway's database.
package database

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"xorm.io/xorm"
	log2 "xorm.io/xorm/log"
	"xorm.io/xorm/names"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/service"
)

const (
	// ServiceName is the name of the gatewayd database service.
	ServiceName = "Database"

	aesKeySize = 32
)

//nolint:gochecknoglobals // global var is used by design
var (
	// GCM is the Galois Counter Mode cipher used to encrypt external accounts passwords.
	GCM cipher.AEAD

	// Owner is the name of the gateway instance specified in the configuration file.
	Owner string

	errUnsuportedDB = errors.New("unsupported database")
)

// DB is the database service. It encapsulates a data connection and implements
// Accessor.
type DB struct {
	// The gateway configuration
	Conf *conf.ServerConfig

	// The service Logger
	logger *log.Logger
	// The state of the database service
	state service.State

	// The database accessor.
	*Standalone
}

func (db *DB) loadAESKey() error {
	if GCM != nil {
		return nil
	}

	filename := db.Conf.Database.AESPassphrase
	if _, err := os.Stat(filepath.FromSlash(filename)); os.IsNotExist(err) {
		db.logger.Infof("Creating AES passphrase file at '%s'", filename)

		key := make([]byte, aesKeySize)

		if _, err := rand.Read(key); err != nil {
			return fmt.Errorf("cannot generate AES key: %w", err)
		}

		if err := ioutil.WriteFile(filepath.FromSlash(filename), key, 0o600); err != nil {
			return fmt.Errorf("cannot write AES key to file %q: %w", filename, err)
		}
	}

	key, err := ioutil.ReadFile(filepath.FromSlash(filename))
	if err != nil {
		return fmt.Errorf("cannot read AES key from file %q: %w", filename, err)
	}

	c, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("cannot initialize AES key: %w", err)
	}

	GCM, err = cipher.NewGCM(c)
	if err != nil {
		return fmt.Errorf("cannot initialize AES key: %w", err)
	}

	return nil
}

// createDSN creates and returns the dataSourceName string necessary to open
// a connection to the database. The DSN varies depending on the options given
// in the database configuration.
func (db *DB) createConnectionInfo() (string, string, func(*xorm.Engine) error, error) {
	rdbms := db.Conf.Database.Type

	info, ok := supportedRBMS[rdbms]
	if !ok {
		return "", "", nil, fmt.Errorf("unknown database type '%s': %w", rdbms, errUnsuportedDB)
	}

	driver, dsn, f := info(&db.Conf.Database)

	return driver, dsn, f, nil
}

type dbinfo func(*conf.DatabaseConfig) (string, string, func(*xorm.Engine) error)

//nolint:gochecknoglobals // global var is used by design
var supportedRBMS = map[string]dbinfo{}

func (db *DB) initEngine() (*xorm.Engine, error) {
	driver, dsn, init, err := db.createConnectionInfo()
	if err != nil {
		db.logger.Criticalf("Database configuration invalid: %s", err)

		return nil, err
	}

	engine, err := xorm.NewEngine(driver, dsn)
	if err != nil {
		db.logger.Criticalf("Failed to open database: %s", err)

		return nil, fmt.Errorf("cannot initialize database access: %w", err)
	}

	engine.SetLogger(log2.DiscardLogger{})
	engine.SetMapper(names.GonicMapper{})

	if err := init(engine); err != nil {
		return nil, err
	}

	if err := engine.Ping(); err != nil {
		db.logger.Errorf("Failed to access database: %s", err)

		return nil, fmt.Errorf("cannot access database: %w", err)
	}

	return engine, nil
}

// Start launches the database service using the configuration given in the
// Environment field. If the configuration in invalid, or if the database
// cannot be reached, an error is returned.
// If the service is already running, this function does nothing.
func (db *DB) Start() error {
	if db.logger == nil {
		db.logger = log.NewLogger(ServiceName)
	}

	db.logger.Info("Starting database service...")

	if code, _ := db.state.Get(); code != service.Offline && code != service.Error {
		db.logger.Info("Service is already running")

		return nil
	}

	db.state.Set(service.Starting, "")
	Owner = db.Conf.GatewayName

	if err := db.loadAESKey(); err != nil {
		db.state.Set(service.Error, err.Error())
		db.logger.Criticalf("Failed to load AES key: %s", err)

		return err
	}

	engine, err := db.initEngine()
	if err != nil {
		db.state.Set(service.Error, err.Error())

		return err
	}

	db.Standalone = &Standalone{
		engine: engine,
		logger: db.logger,
		conf:   &db.Conf.Database,
	}

	if err := initTables(db.Standalone); err != nil {
		if err2 := engine.Close(); err2 != nil {
			db.logger.Warningf("an error occurred while closing the database: %v", err2)
		}

		db.state.Set(service.Error, err.Error())
		db.logger.Errorf("Failed to create tables: %s", err)

		return err
	}

	if err := db.checkVersion(); err != nil {
		if err2 := engine.Close(); err2 != nil {
			db.logger.Warningf("an error occurred while closing the database: %v", err2)
		}

		db.state.Set(service.Error, err.Error())

		return err
	}

	db.logger.Info("Startup successful")
	db.state.Set(service.Running, "")

	return nil
}

// Stop shuts down the database service. If an error occurred during the shutdown,
// an error is returned.
// If the service is not running, this function does nothing.
func (db *DB) Stop(_ context.Context) error {
	db.logger.Info("Shutting down...")

	if code, _ := db.state.Get(); code != service.Running {
		db.logger.Info("Service is already offline")

		return nil
	}

	db.state.Set(service.ShuttingDown, "")

	err := db.Standalone.engine.Close()
	if err != nil {
		db.state.Set(service.Error, err.Error())
		db.logger.Infof("Error while closing the database: %s", err)

		return fmt.Errorf("an error occurred while closing the database: %w", err)
	}

	db.state.Set(service.Offline, "")
	db.logger.Info("Shutdown complete")
	db.Standalone.engine = nil

	return nil
}

// State returns the state of the database service.
func (db *DB) State() *service.State {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	//nolint:errcheck //thiserror is handled another inside ping
	_ = ping(&db.state, db.Standalone.engine.Context(ctx), db.logger)

	return &db.state
}
