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
	"sync"

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
	// ErrServiceUnavailable is the error returned by database operation
	// methods when the database is inactive
	ErrServiceUnavailable = errors.New("the database service is not running")

	// ErrNilRecord is the error returned by database operation when the object
	// Which should be  used to generate the query or used to unmarshal the
	// query result is nil
	ErrNilRecord = errors.New("the record cannot be nil")

	// ErrNotFound is the error returned by the Get operations when the queried
	// row is not found in database
	ErrNotFound = errors.New("the record does not exist")
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
	Get(bean interface{}) error
	Select(bean interface{}, filters *Filters) error
	Create(bean interface{}) error
	Update(bean interface{}, id uint64, isReplace bool) error
	Delete(bean interface{}) error
	Exists(bean interface{}) (bool, error)
	Execute(sqlorArgs ...interface{}) error
	Query(sqlorArgs ...interface{}) ([]map[string]interface{}, error)
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
	// The mutex used for the test database
	testDBLock *sync.Mutex
}

func loadAESKey(filename string) error {

	if _, err := os.Stat(filepath.FromSlash(filename)); os.IsNotExist(err) {
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

	if err := loadAESKey(db.Conf.Database.AESPassphrase); err != nil {
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
func ping(state *service.State, db xorm.Interface, logger *log.Logger) error {
	if err := db.Ping(); err != nil {
		logger.Errorf(err.Error())
		state.Set(service.Error, err.Error())
		return err
	}
	state.Set(service.Running, "")
	return nil
}

// State returns the state of the database service.
func (db *DB) State() *service.State {
	_ = ping(&db.state, db.engine, db.logger)
	return &db.state
}

// NewQuery returns new query builder.
func (db *DB) NewQuery() *builder.Builder {
	return builder.Dialect(string(db.engine.Dialect().DBType()))
}

// Get retrieves one record from the database and fills the bean with it. Non-empty
// fields are used as conditions.
func (db *DB) Get(bean interface{}) error {
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
func (db *DB) Create(bean interface{}) error {
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
func (db *DB) Update(bean interface{}, id uint64, isReplace bool) error {
	db.logger.Debugf("Update requested with %#v", bean)

	ses, err := db.BeginTransaction()
	if err != nil {
		return err
	}

	if err := ses.Update(bean, id, isReplace); err != nil {
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
func (db *DB) Delete(bean interface{}) error {
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

// Exists checks if the given record exists in the database. If the database
// cannot be queried, an error is returned.
func (db *DB) Exists(bean interface{}) (bool, error) {
	db.logger.Debugf("Exists requested with %#v", bean)

	ses, err := db.BeginTransaction()
	if err != nil {
		return false, err
	}

	exist, err := ses.Exists(bean)
	if err != nil {
		ses.Rollback()
		return false, err
	}
	if err := ses.Commit(); err != nil {
		return false, err
	}
	return exist, nil
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

	if db.testDBLock != nil {
		db.testDBLock.Lock()
		defer func() {
			if err != nil {
				db.testDBLock.Unlock()
			}
		}()
	}

	s := db.engine.NewSession()
	if err = s.Begin(); err != nil {
		if pErr := ping(&db.state, db.engine, db.logger); pErr != nil {
			err = pErr
			return
		}
		return
	}
	ses = &Session{
		session:    s,
		logger:     db.logger,
		state:      &db.state,
		testDBLock: db.testDBLock,
	}
	db.logger.Debug("Transaction started")

	return ses, err
}

// Session is a struct used to perform transactions on the database. A session
// can be created with the `BeginTransaction` method. Once the transaction is
// complete, it can be committed using `Commit`, it can also be canceled using
// the `Rollback` function.
type Session struct {
	session    *xorm.Session
	logger     *log.Logger
	state      *service.State
	testDBLock *sync.Mutex
}

// Get adds a 'get' query to the transaction. If the query cannot be executed,
// an error is returned.
func (s *Session) Get(bean interface{}) error {
	s.logger.Debugf("Transaction 'Get' with %#v", bean)
	if s, _ := s.state.Get(); s != service.Running {
		return ErrServiceUnavailable
	}
	if bean == nil {
		return ErrNilRecord
	}

	if has, err := s.session.Get(bean); err != nil {
		if err := ping(s.state, s.session, s.logger); err != nil {
			return err
		}
		return err
	} else if !has {
		return ErrNotFound
	}
	return nil
}

// Select adds a 'select' query to the transaction with the conditions given in
// the `filters` parameter.  If the query cannot be executed, an error is returned.
func (s *Session) Select(bean interface{}, filters *Filters) error {
	s.logger.Debugf("Transaction 'Select' with %#v", bean)
	if s, _ := s.state.Get(); s != service.Running {
		return ErrServiceUnavailable
	}
	if bean == nil {
		return ErrNilRecord
	}
	var query xorm.Interface = s.session
	if filters != nil {
		if filters.Conditions != nil {
			query = query.Where(filters.Conditions)
		}
		query = query.Limit(filters.Limit, filters.Offset).OrderBy(filters.Order)
	}

	if err := query.Find(bean); err != nil {
		if err := ping(s.state, s.session, s.logger); err != nil {
			return err
		}
		return err
	}
	return nil
}

// Create adds an 'insert' query to the transaction. If the query cannot be executed,
// an error is returned.
func (s *Session) Create(bean interface{}) error {
	s.logger.Debugf("Transaction 'Create' with %#v", bean)
	if s, _ := s.state.Get(); s != service.Running {
		return ErrServiceUnavailable
	}
	if bean == nil {
		return ErrNilRecord
	}

	exec := func() error {
		if hook, ok := bean.(insertHook); ok {
			if err := hook.BeforeInsert(s); err != nil {
				return err
			}
		}

		if _, err := s.session.InsertOne(bean); err != nil {
			return err
		}

		return nil
	}

	if err := exec(); err != nil {
		if err := ping(s.state, s.session, s.logger); err != nil {
			return err
		}
		return err
	}
	return nil
}

// Update adds an 'update' query to the transaction. If the query cannot be executed,
// an error is returned.
func (s *Session) Update(bean interface{}, id uint64, isReplace bool) error {
	s.logger.Debugf("Transaction 'Update' with %#v", bean)
	if s, _ := s.state.Get(); s != service.Running {
		return ErrServiceUnavailable
	}
	if bean == nil {
		return ErrNilRecord
	}
	if t, ok := bean.(xorm.TableName); ok {
		query := builder.Select("id").From(t.TableName()).Where(builder.Eq{"id": id})
		if res, err := s.session.Query(query); err != nil {
			return err
		} else if len(res) == 0 {
			return ErrNotFound
		}
	}

	exec := func() error {
		if hook, ok := bean.(updateHook); ok {
			if err := hook.BeforeUpdate(s, id); err != nil {
				return err
			}
		}

		if isReplace {
			if _, err := s.session.AllCols().ID(id).Update(bean); err != nil {
				return err
			}
		}
		if _, err := s.session.ID(id).Update(bean); err != nil {
			return err
		}

		return nil
	}

	if err := exec(); err != nil {
		if err := ping(s.state, s.session, s.logger); err != nil {
			return err
		}
		return err
	}
	return nil
}

// Delete adds an 'delete' query to the transaction. If the query cannot be executed,
// an error is returned.
func (s *Session) Delete(bean interface{}) error {
	s.logger.Debugf("Transaction 'Delete' with %#v", bean)
	if s, _ := s.state.Get(); s != service.Running {
		return ErrServiceUnavailable
	}
	if bean == nil {
		return ErrNilRecord
	}
	if exist, err := s.Exists(bean); err != nil {
		return err
	} else if !exist {
		return ErrNotFound
	}

	exec := func() error {
		if hook, ok := bean.(deleteHook); ok {
			if err := hook.BeforeDelete(s); err != nil {
				return err
			}
		}

		if _, err := s.session.Delete(bean); err != nil {
			return err
		}

		return nil
	}

	if err := exec(); err != nil {
		if err := ping(s.state, s.session, s.logger); err != nil {
			return err
		}
		return err
	}
	return nil
}

// Exists adds an 'exist' query to the transaction. If the query cannot be executed,
// an error is returned.
func (s *Session) Exists(bean interface{}) (bool, error) {
	s.logger.Debugf("Transaction 'Exists' with %#v", bean)
	if s, _ := s.state.Get(); s != service.Running {
		return false, ErrServiceUnavailable
	}
	if bean == nil {
		return false, ErrNilRecord
	}

	has, err := s.session.Exist(bean)
	if err != nil {
		if err := ping(s.state, s.session, s.logger); err != nil {
			return false, err
		}
		return false, err
	}
	return has, nil
}

// Execute adds a custom raw query to the transaction. If the query cannot be executed,
// an error is returned. If the command must return a result, use `Query` instead.
func (s *Session) Execute(sqlorArgs ...interface{}) error {
	s.logger.Debugf("Transaction 'Execute' with %#v", sqlorArgs)
	if s, _ := s.state.Get(); s != service.Running {
		return ErrServiceUnavailable
	}

	if _, err := s.session.Exec(sqlorArgs...); err != nil {
		if err := ping(s.state, s.session, s.logger); err != nil {
			return err
		}
		return err
	}
	return nil
}

// Query adds a custom raw query to the transaction. If the query cannot be executed,
// an error is returned. The function returns a slice of map[string]interface{}
// which contains the result of the query.
func (s *Session) Query(sqlorArgs ...interface{}) ([]map[string]interface{}, error) {
	s.logger.Debugf("Transaction 'Execute' with %#v", sqlorArgs)
	if s, _ := s.state.Get(); s != service.Running {
		return nil, ErrServiceUnavailable
	}

	res, err := s.session.QueryInterface(sqlorArgs...)
	if err != nil {
		if err := ping(s.state, s.session, s.logger); err != nil {
			return nil, err
		}
		return nil, err
	}
	return res, nil
}

// Rollback cancels the transaction, and rolls back any changes made to the
// database. When this function is called, the session is closed, which means
// it cannot be used to perform any more transactions.
func (s *Session) Rollback() {
	defer func() {
		if s.testDBLock != nil {
			s.testDBLock.Unlock()
		}
	}()

	s.logger.Debugf("Rolling back changes %v", s)
	s.session.Close()
}

// Commit commits all the transactions pending operations to the database. If
// the commit fails, the changes are dropped, and an error is returned. After
// this function is returned, the session is closed and no more transactions can
// be performed using this instance.
func (s *Session) Commit() error {
	s.logger.Debug("Committing transaction")
	defer func() {
		if s.testDBLock != nil {
			s.testDBLock.Unlock()
		}
		s.session.Close()
	}()

	if st, _ := s.state.Get(); st != service.Running {
		return ErrServiceUnavailable
	}
	err := s.session.Commit()
	if err != nil {
		s.logger.Errorf("Commit failed (%s), changes were not committed", err)
		if err := ping(s.state, s.session, s.logger); err != nil {
			return err
		}
		return err
	}
	return nil
}
