// Package database contains the module for accessing the gateway's database.
package database

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
	"xorm.io/xorm"
	log2 "xorm.io/xorm/log"
	"xorm.io/xorm/names"
)

const (
	// ServiceName is the name of the gatewayd database service
	ServiceName = "Database"
)

var (
	// GCM is the Galois Counter Mode cipher used to encrypt external accounts passwords.
	GCM cipher.AEAD

	// Owner is the name of the gateway instance specified in the configuration file.
	Owner string
)

// DB is the database service. It encapsulates a data connection and implements
// Accessor
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
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return err
		}

		if err := ioutil.WriteFile(filepath.FromSlash(filename), key, 0600); err != nil {
			return err
		}
	}

	key, err := ioutil.ReadFile(filepath.FromSlash(filename))
	if err != nil {
		return err
	}

	c, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	GCM, err = cipher.NewGCM(c)
	if err != nil {
		return err
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
		return "", "", nil, fmt.Errorf("unknown database type '%s'", rdbms)
	}

	driver, dsn, f := info(db.Conf.Database)
	return driver, dsn, f, nil
}

type dbinfo func(conf.DatabaseConfig) (string, string, func(*xorm.Engine) error)

var supportedRBMS = map[string]dbinfo{}

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

	driver, dsn, init, err := db.createConnectionInfo()
	if err != nil {
		db.state.Set(service.Error, err.Error())
		db.logger.Criticalf("Database configuration invalid: %s", err)
		return err
	}

	engine, err := xorm.NewEngine(driver, dsn)
	if err != nil {
		db.state.Set(service.Error, err.Error())
		db.logger.Criticalf("Failed to open database: %s", err)
		return err
	}
	engine.SetLogger(log2.DiscardLogger{})
	engine.SetMapper(names.GonicMapper{})
	if err := init(engine); err != nil {
		return err
	}

	if err := engine.Ping(); err != nil {
		db.state.Set(service.Error, err.Error())
		db.logger.Errorf("Failed to access database: %s", err)
		return err
	}

	db.state.Set(service.Running, "")

	db.Standalone = &Standalone{
		engine: engine,
		logger: db.logger,
	}

	if err := initTables(db.Standalone); err != nil {
		db.state.Set(service.Error, err.Error())
		db.logger.Errorf("Failed to create tables: %s", err)
		return err
	}

	db.logger.Info("Startup successful")

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
		return err
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
	_ = ping(&db.state, db.Standalone.engine.Context(ctx), db.logger)
	return &db.state
}
