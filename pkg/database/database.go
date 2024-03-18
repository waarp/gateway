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
	"time"

	"code.waarp.fr/lib/log"
	"xorm.io/xorm"
	xNames "xorm.io/xorm/names"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/names"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/state"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

const aesKeySize = 32

//nolint:gochecknoglobals // global var is used by design
var (
	// GCM is the Galois Counter Mode cipher used to encrypt external accounts passwords.
	GCM cipher.AEAD

	errUnsupportedDB = errors.New("unsupported database")
)

// DB is the database service. It encapsulates a data connection and implements
// Accessor.
type DB struct {
	// The service Logger
	logger *log.Logger
	// The state of the database service
	state state.State

	// The database accessor.
	*Standalone
}

func (db *DB) loadAESKey() error {
	if GCM != nil {
		return nil
	}

	filename := conf.GlobalConfig.Database.AESPassphrase
	if _, err := os.Stat(utils.ToOSPath(filename)); os.IsNotExist(err) {
		db.logger.Info("Creating AES passphrase file at '%s'", filename)

		key := make([]byte, aesKeySize)

		if _, err := rand.Read(key); err != nil {
			return fmt.Errorf("cannot generate AES key: %w", err)
		}

		if err := os.WriteFile(utils.ToOSPath(filename), key, 0o600); err != nil {
			return fmt.Errorf("cannot write AES key to file %q: %w", filename, err)
		}
	}

	key, err := os.ReadFile(utils.ToOSPath(filename))
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

// createConnectionInfo creates and returns the dataSourceName string necessary
// to open a connection to the database, along with the driver and an optional
// initialisation function. The DSN varies depending on the options given
// in the database configuration.
func (db *DB) createConnectionInfo() (*DBInfo, error) {
	rdbms := conf.GlobalConfig.Database.Type

	makeConnInfo, ok := supportedRBMS[rdbms]
	if !ok {
		return nil, fmt.Errorf("unknown database type '%s': %w", rdbms, errUnsupportedDB)
	}

	return makeConnInfo(), nil
}

type DBInfo struct {
	Driver, DSN string
	ConnLimit   int
}

//nolint:gochecknoglobals // global var is used by design
var supportedRBMS = map[string]func() *DBInfo{}

func AddRDBMS(rdbms string, makeConnInfo func() *DBInfo) {
	supportedRBMS[rdbms] = makeConnInfo
}

func (db *DB) initEngine() (*xorm.Engine, error) {
	connInfo, err := db.createConnectionInfo()
	if err != nil {
		db.logger.Critical("Database configuration invalid: %s", err)

		return nil, err
	}

	engine, err := xorm.NewEngine(connInfo.Driver, connInfo.DSN)
	if err != nil {
		db.logger.Critical("Failed to open database: %s", err)

		return nil, fmt.Errorf("cannot initialize database access: %w", err)
	}

	db.setLogger(engine)
	engine.SetMapper(xNames.GonicMapper{})

	if err := engine.Ping(); err != nil {
		db.logger.Error("Failed to access database: %s", err)

		return nil, fmt.Errorf("cannot access database: %w", err)
	}

	if connInfo.ConnLimit > 0 {
		engine.SetMaxOpenConns(connInfo.ConnLimit)
	}

	return engine, nil
}

// Start launches the database service using the configuration given in the
// Environment field. If the configuration in invalid, or if the database
// cannot be reached, an error is returned.
// If the service is already running, this function does nothing.
func (db *DB) Start() error {
	return db.start(true)
}

func (db *DB) start(withInit bool) error {
	if db.logger == nil {
		db.logger = conf.GetLogger(names.DatabaseServiceName)
	}

	db.logger.Info("Starting database service...")

	if code, _ := db.state.Get(); code != state.Offline && code != state.Error {
		db.logger.Info("Service is already running")

		return nil
	}

	db.state.Set(state.Starting, "")

	if err := db.loadAESKey(); err != nil {
		db.state.Set(state.Error, err.Error())
		db.logger.Critical("Failed to load AES key: %s", err)

		return err
	}

	engine, err := db.initEngine()
	if err != nil {
		db.state.Set(state.Error, err.Error())

		return err
	}

	db.Standalone = &Standalone{
		engine: engine,
		logger: db.logger,
	}

	if err1 := db.checkVersion(); err1 != nil {
		if err2 := engine.Close(); err2 != nil {
			db.logger.Warning("an error occurred while closing the database: %v", err2)
		}

		db.state.Set(state.Error, err1.Error())

		return err1
	}

	if withInit {
		if err1 := initDatabase(db.Standalone); err1 != nil {
			if err2 := engine.Close(); err2 != nil {
				db.logger.Warning("an error occurred while closing the database: %v", err2)
			}

			db.state.Set(state.Error, err1.Error())

			return err1
		}
	}

	db.logger.Info("Startup successful")
	db.state.Set(state.Running, "")

	return nil
}

// Stop shuts down the database service. If an error occurred during the shutdown,
// an error is returned.
// If the service is not running, this function does nothing.
func (db *DB) Stop(ctx context.Context) error {
	db.logger.Info("Shutting down...")

	if code, _ := db.state.Get(); code != state.Running {
		db.logger.Info("Service is already offline")

		return nil
	}

	db.state.Set(state.ShuttingDown, "")

	if err := db.Standalone.engine.Close(); err != nil {
		db.state.Set(state.Error, err.Error())
		db.logger.Info("Error while closing the database: %s", err)

		return fmt.Errorf("an error occurred while closing the database: %w", err)
	}

	select {
	case <-ctx.Done():
		db.logger.Warning("Failed to close the pending transactions")
		db.logger.Warning("Force closing the database")
	case <-func() chan bool {
		done := make(chan bool)

		db.sessions.Range(func(_, ses any) bool {
			//nolint:forcetypeassert //type assert will always succeed
			if err := ses.(*Session).session.Close(); err != nil {
				db.logger.Warning("Failed to close session: %v", err)
			}

			return true
		})

		close(done)

		return done
	}():
	}

	db.state.Set(state.Offline, "")
	db.logger.Info("Shutdown complete")
	db.Standalone.engine = nil

	return nil
}

// State returns the state of the database service.
func (db *DB) State() *state.State {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if db.engine != nil {
		//nolint:errcheck //this error is handled another inside ping
		_ = ping(&db.state, db.Standalone.engine.Context(ctx), db.logger)
	}

	return &db.state
}
