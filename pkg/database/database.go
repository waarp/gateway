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

	"code.waarp.fr/lib/log"
	"github.com/puzpuzpuz/xsync/v4"
	"xorm.io/xorm"
	xnames "xorm.io/xorm/names"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	aesKeySize = 32

	ServiceName = "Database"
)

//nolint:gochecknoglobals // global var is used by design
var (
	// GCM is the Galois Counter Mode cipher used to encrypt external accounts passwords.
	GCM cipher.AEAD

	errUnsupportedDB = errors.New("unsupported database")
)

// DB is the database service. It encapsulates a data connection and implements
// Accessor.
type DB struct {
	engine   *xorm.Engine
	sessions *xsync.Map[int64, *Session]

	// The service Logger
	logger *log.Logger
	// The state of the database service
	state utils.State
}

func (db *DB) loadAESKey() error {
	if GCM != nil {
		return nil
	}

	filename := conf.GlobalConfig.Database.AESPassphrase
	if _, statErr := os.Stat(filepath.Clean(filename)); os.IsNotExist(statErr) {
		db.logger.Infof("Creating AES passphrase file at %q", filename)

		key := make([]byte, aesKeySize)

		if _, err := rand.Read(key); err != nil {
			return fmt.Errorf("cannot generate AES key: %w", err)
		}

		if err := os.WriteFile(filepath.Clean(filename), key, 0o600); err != nil {
			return fmt.Errorf("cannot write AES key to file %q: %w", filename, err)
		}
	}

	var gcmErr error
	GCM, gcmErr = NewGCM(filename)

	return gcmErr
}

func NewGCM(filename string) (cipher.AEAD, error) {
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

// createConnectionInfo creates and returns the dataSourceName string necessary
// to open a connection to the database, along with the driver and an optional
// initialisation function. The DSN varies depending on the options given
// in the database configuration.
func (db *DB) createConnectionInfo() (*DBInfo, error) {
	rdbms := conf.GlobalConfig.Database.Type

	makeConnInfo, ok := SupportedRBMS[rdbms]
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
var SupportedRBMS = map[string]func() *DBInfo{}

func (db *DB) initEngine() error {
	connInfo, err := db.createConnectionInfo()
	if err != nil {
		db.logger.Criticalf("Database configuration invalid: %v", err)

		return err
	}

	engine, err := xorm.NewEngine(connInfo.Driver, connInfo.DSN)
	if err != nil {
		db.logger.Criticalf("Failed to open database: %v", err)

		return fmt.Errorf("cannot initialize database access: %w", err)
	}

	db.setLogger(engine)
	engine.SetMapper(xnames.GonicMapper{})

	if err = engine.Ping(); err != nil {
		db.logger.Errorf("Failed to access database: %v", err)

		return fmt.Errorf("cannot access database: %w", err)
	}

	if connInfo.ConnLimit > 0 {
		engine.SetMaxOpenConns(connInfo.ConnLimit)
	}

	db.engine = engine
	db.sessions = xsync.NewMap[int64, *Session]()

	return nil
}

// Start launches the database service using the configuration given in the
// Environment field. If the configuration in invalid, or if the database
// cannot be reached, an error is returned.
// If the service is already running, this function does nothing.
func (db *DB) Start() error {
	if db.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	if err := db.start(true); err != nil {
		db.state.Set(utils.StateError, err.Error())

		return err
	}

	db.state.Set(utils.StateRunning, "")

	return nil
}

func (db *DB) start(withInit bool) error {
	db.logger = logging.NewLogger(ServiceName)
	db.logger.Info("Starting database service...")

	if err := db.loadAESKey(); err != nil {
		db.logger.Criticalf("Failed to load AES key: %v", err)

		return err
	}

	if err := db.initEngine(); err != nil {
		return err
	}

	if err1 := db.checkVersion(); err1 != nil {
		if err2 := db.engine.Close(); err2 != nil {
			db.logger.Warningf("an error occurred while closing the database: %v", err2)
		}

		return err1
	}

	if withInit {
		if err1 := db.initDatabase(); err1 != nil {
			if err2 := db.engine.Close(); err2 != nil {
				db.logger.Warningf("an error occurred while closing the database: %v", err2)
			}

			return err1
		}
	}

	db.logger.Info("Startup successful")

	return nil
}

// Stop shuts down the database service. If an error occurred during the shutdown,
// an error is returned.
// If the service is not running, this function does nothing.
func (db *DB) Stop(ctx context.Context) error {
	if !db.state.IsRunning() {
		return utils.ErrNotRunning
	}

	if err := db.stop(ctx); err != nil {
		db.state.Set(utils.StateError, err.Error())

		return err
	}

	db.state.Set(utils.StateOffline, "")

	return nil
}

func (db *DB) stop(ctx context.Context) error {
	defer func() { db.engine = nil }()

	db.logger.Info("Shutting down...")

	if err := db.engine.Close(); err != nil {
		db.logger.Infof("Error while closing the database: %v", err)

		return fmt.Errorf("an error occurred while closing the database: %w", err)
	}

	select {
	case <-ctx.Done():
		db.logger.Warning("Failed to close the pending transactions")
		db.logger.Warning("Force closing the database")
	case <-func() chan bool {
		done := make(chan bool)

		db.sessions.Range(func(_ int64, ses *Session) bool {
			if err := ses.session.Close(); err != nil {
				db.logger.Warningf("Failed to close session: %v", err)
			}

			return true
		})

		close(done)

		return done
	}():
	}

	db.logger.Info("Shutdown complete")
	db.engine = nil

	return nil
}

// State returns the state of the database service.
func (db *DB) State() (utils.StateCode, string) {
	return db.state.Get()
}
