// Package database contains the module for accessing the gateway's database.
package database

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
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
// Deprecated:
type Accessor interface {
	// Deprecated:
	Get(tableName) error
	// Deprecated:
	Select(interface{}, *Filters) error
	// Deprecated:
	Create(tableName) error
	// Deprecated:
	Update(entry) error
	// Deprecated:
	Delete(tableName) error
	// Deprecated:
	Execute(...interface{}) error
	// Deprecated:
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

// Transaction executes all the commands in the given function as a transaction.
// The transaction will be then be roll-backed or committed, depending whether
// the function returned an error or not.
func (db *DB) Transaction(f func(*Session) error) error {
	db.logger.Debug("Beginning transaction")
	if err := db.transaction(f); err != nil {
		db.logger.Debug("Transaction failed, changes have been rolled back")
		return err
	}
	db.logger.Debug("Transaction succeeded, changes have been committed")
	return nil
}

func (db *DB) transaction(f func(*Session) error) error {
	ses := db.newSession()
	defer ses.rollback()

	if err := ses.session.Begin(); err != nil {
		return err
	}

	if err := f(ses); err != nil {
		return err
	}

	return ses.commit()
}

// Exec executes the given SQL command. The command can be created using the
// various command builder functions available in this package, such as:
// - Insert
// - Update
// - Delete
func (db *DB) Exec(cmd command) (sql.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var res sql.Result
	err := db.transaction(func(ses *Session) error {
		ses.session.Context(ctx)
		var err1 error
		res, err1 = ses.Exec(cmd)
		return err1
	})
	return res, err
}

// Query2 executes the given SQL query and returns the selected database rows
// as an Iterator. The query can be created using the various query builder
// functions available in this package, such as:
// - Select
func (db *DB) Query2(query query) (Iterator, error) {
	s := db.newSession()
	it, err := s.Query2(query)
	return &saIterator{it, s.session.Close}, err
}

// Count is a wrapper function which counts the number of rows returned by a
// 'SELECT' query without having to iterate on all the rows.
func (db *DB) Count(query *selectQuery) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	s := db.newSession()
	s.session.Context(ctx)
	return s.Count(query)
}

// NewQuery returns new query builder.
// Deprecated:
func (db *DB) NewQuery() *builder.Builder {
	return builder.Dialect(string(db.engine.Dialect().DBType()))
}

// Get retrieves one record from the database and fills the bean with it. Non-empty
// fields are used as conditions.
// Deprecated:
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
// Deprecated:
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
// Deprecated:
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
// Deprecated:
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
// Deprecated:
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
// Deprecated:
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
// Deprecated:
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
// Deprecated:
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
