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
	"github.com/go-xorm/builder"
	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"
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

// Accessor is the interface that lists the method sets needed to query a
// database.
type Accessor interface {
	Get(tableName) error
	Select(interface{}, *Filters) error
	Create(tableName) error
	Update(entry) error
	Delete(tableName) error
	Execute(...interface{}) error
	Query(...interface{}) ([]map[string]interface{}, error)
}

// DB is the database service. It encapsulates a data connection and implements
// Accessor
type DB struct {
	// The gateway configuration
	Conf *conf.ServerConfig

	// The service logger
	logger *log.Logger
	// The state of the database service
	state service.State
	// The Xorm engine handling the database requests
	engine *xorm.Engine
	// The name of the SQL database driver used by the engine
	driverName string
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
func (db *DB) createConnectionInfo() (driver string, dsn string, err error) {
	rdbms := db.Conf.Database.Type

	info, ok := supportedRBMS[rdbms]
	if !ok {
		return "", "", fmt.Errorf("unknown database type '%s'", rdbms)
	}

	driver, dsn = info(db.Conf.Database)
	return
}

type dbinfo func(conf.DatabaseConfig) (string, string)

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

	driver, dsn, err := db.createConnectionInfo()
	if err != nil {
		db.state.Set(service.Error, err.Error())
		db.logger.Criticalf("Database configuration invalid: %s", err)
		return err
	}

	db.driverName = driver
	engine, err := xorm.NewEngine(db.driverName, dsn)
	if err != nil {
		db.state.Set(service.Error, err.Error())
		db.logger.Criticalf("Failed to open database: %s", err)
		return err
	}
	db.engine = engine
	db.engine.SetLogLevel(core.LOG_WARNING)
	db.engine.SetMapper(core.GonicMapper{})

	if err := db.engine.Ping(); err != nil {
		db.state.Set(service.Error, err.Error())
		db.logger.Errorf("Failed to access database: %s", err)
		return err
	}

	db.state.Set(service.Running, "")

	if err := initTables(db); err != nil {
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

	err := db.engine.Close()
	if err != nil {
		db.state.Set(service.Error, err.Error())
		db.logger.Infof("Error while closing the database: %s", err)
		return err
	}
	db.state.Set(service.Offline, "")
	db.logger.Info("Shutdown complete")
	db.engine = nil
	return nil
}

// ping checks if the database is reachable and updates the service state accordingly.
// If an error occurs while contacting the database, that error is returned.
func ping(state *service.State, ses *xorm.Session, logger *log.Logger) error {
	if err := ses.Ping(); err != nil {
		err = NewInternalError(err, "could not reach database")
		logger.Errorf(err.Error())
		state.Set(service.Error, err.Error())
		return err
	}
	state.Set(service.Running, "")
	return nil
}

// State returns the state of the database service.
func (db *DB) State() *service.State {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = ping(&db.state, db.engine.Context(ctx), db.logger)
	return &db.state
}

// NewQuery returns new query builder.
func (db *DB) NewQuery() *builder.Builder {
	return builder.Dialect(string(db.engine.Dialect().DBType()))
}

// Get retrieves one record from the database and fills the bean with it. Non-empty
// fields are used as conditions.
func (db *DB) Get(bean tableName) error {
	db.logger.Debugf("Get requested with %#v", bean)

	ses, err := db.BeginTransaction()
	if err != nil {
		return err
	}

	if err := ses.Get(bean); err != nil {
		ses.Rollback()
		return err
	}
	if err := ses.Commit(); err != nil {
		return err
	}
	return nil
}

// Select retrieves multiple records from the database using the given filters
// and fills the bean with it. The bean should be of type []Struct or []*Struct,
// and it should be empty.
func (db *DB) Select(bean interface{}, filters *Filters) error {
	db.logger.Debugf("Select requested with %#v", bean)

	ses, err := db.BeginTransaction()
	if err != nil {
		return err
	}

	if err := ses.Select(bean, filters); err != nil {
		ses.Rollback()
		return err
	}
	if err := ses.Commit(); err != nil {
		return err
	}
	return nil
}

// Create inserts the given bean in the database. If the struct cannot be inserted,
// the function returns an error.
func (db *DB) Create(bean tableName) error {
	db.logger.Debugf("Create requested with %#v", bean)

	ses, err := db.BeginTransaction()
	if err != nil {
		return err
	}

	if err := ses.Create(bean); err != nil {
		ses.Rollback()
		return err
	}
	if err := ses.Commit(); err != nil {
		return err
	}
	return nil
}

// Update updates the given bean in the database. If the struct cannot be updated,
// the function returns an error.
func (db *DB) Update(bean entry) error {
	db.logger.Debugf("Update requested with %#v", bean)

	ses, err := db.BeginTransaction()
	if err != nil {
		return err
	}

	if err := ses.Update(bean); err != nil {
		ses.Rollback()
		return err
	}
	if err := ses.Commit(); err != nil {
		return err
	}
	return nil
}

// Delete deletes the given bean from the database. If the record cannot be deleted,
// an error is returned.
func (db *DB) Delete(bean tableName) error {
	db.logger.Debugf("Delete requested with %#v", bean)

	ses, err := db.BeginTransaction()
	if err != nil {
		return err
	}

	if err := ses.Delete(bean); err != nil {
		ses.Rollback()
		return err
	}
	if err := ses.Commit(); err != nil {
		return err
	}
	return nil
}

// Execute executes the given SQL command. The command can be a raw string with
// arguments, or an xorm.Builder struct.
func (db *DB) Execute(sqlOrArgs ...interface{}) error {
	db.logger.Debugf("Execute requested with %#v", sqlOrArgs)

	ses, err := db.BeginTransaction()
	if err != nil {
		return err
	}

	if err := ses.Execute(sqlOrArgs...); err != nil {
		ses.Rollback()
		return err
	}
	if err := ses.Commit(); err != nil {
		return err
	}
	return nil
}

// Query executes the given SQL query and returns the result. The query can be
// a raw string with arguments, or an xorm.Builder struct.
func (db *DB) Query(sqlOrArgs ...interface{}) ([]map[string]interface{}, error) {
	db.logger.Debugf("Query requested with %#v", sqlOrArgs)

	ses, err := db.BeginTransaction()
	if err != nil {
		return nil, err
	}

	res, err := ses.Query(sqlOrArgs...)
	if err != nil {
		ses.Rollback()
		return nil, err
	}
	if err := ses.Commit(); err != nil {
		return nil, err
	}
	return res, nil
}

// BeginTransaction returns a new session on which a database transaction can
// be performed.
func (db *DB) BeginTransaction() (ses *Session, err error) {
	if s, _ := db.state.Get(); s != service.Running {
		return nil, ErrServiceUnavailable
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	s := db.engine.NewSession().Context(ctx)
	if iErr := s.Begin(); iErr != nil {
		if pErr := ping(&db.state, s, db.logger); pErr != nil {
			err = pErr
			return
		}
		err = NewInternalError(iErr, "failed to start database transaction")
		return
	}
	ses = &Session{
		session: s,
		logger:  db.logger,
		state:   &db.state,
		cancel:  cancel,
	}
	db.logger.Debug("Transaction started")

	return ses, err
}
