package database

import (
	"context"
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"
	"github.com/pkg/errors"
)

const (
	// ServiceName is he name of the gatewayd database service
	ServiceName = "Database"
)

var (
	// ErrServiceUnavailable is the error returned by database operation
	// methods when the database is inactive
	ErrServiceUnavailable = errors.New("the database service is not running")

	// ErrNilRecord is the error returned by database operation when the object
	// Which should be  used to generate the query or used to unmarshal the
	// query resultis nil
	ErrNilRecord = errors.New("the record cannot be nil")

	// ErrNotFound is the error returned by the Get operations when the queried
	// row is not found in database
	ErrNotFound = errors.New("the record does not exist")
)

// Accessor is the interface that lists the method sets needed to query a
// database.
type Accessor interface {
	Get(bean interface{}) error
	Select(bean interface{}, filters *Filters) error
	Create(bean interface{}) error
	Update(before, after interface{}) error
	Delete(bean interface{}) error
	Exists(bean interface{}) (bool, error)
	Execute(sqlorArgs ...interface{}) error
	Query(sqlorArgs ...interface{}) ([]map[string]interface{}, error)
}

// Db is the database service. it encasulates a data connection and implements
// Accessor
type Db struct {
	// The service logger
	Logger *log.Logger
	// The gateway configuration
	Conf *conf.ServerConfig
	// The state of the database service
	state service.State
	// The Xorm engine handling the database requests
	engine *xorm.Engine
	// The name of the SQL database driver used by the engine
	driverName string
}

// createDSN creates and returns the dataSourceName string necessary to open
// a connection to the database. The DSN varies depending on the options given
// in the database configuration.
func (db *Db) createConnectionInfo() (driver string, dsn string, err error) {
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
func (db *Db) Start() error {
	if db.Logger == nil {
		db.Logger = log.NewLogger(ServiceName)
	}

	db.Logger.Info("Starting database service...")
	if code, _ := db.state.Get(); code != service.Offline && code != service.Error {
		db.Logger.Info("Service is already running")
		return nil
	}
	db.state.Set(service.Starting, "")

	driver, dsn, err := db.createConnectionInfo()
	if err != nil {
		db.state.Set(service.Error, err.Error())
		db.Logger.Criticalf("Database configuration invalid: %s", err)
		return err
	}

	db.driverName = driver
	engine, err := xorm.NewEngine(db.driverName, dsn)
	if err != nil {
		db.state.Set(service.Error, err.Error())
		db.Logger.Criticalf("Failed to open database: %s", err)
		return err
	}
	db.engine = engine
	db.engine.SetLogLevel(core.LOG_WARNING)
	db.engine.SetMapper(core.GonicMapper{})

	if err := db.engine.Ping(); err != nil {
		db.state.Set(service.Error, err.Error())
		db.Logger.Errorf("Failed to access database: %s", err)
		return err
	}

	if err := initTables(db); err != nil {
		db.Logger.Errorf("Failed to create tables: %s", err)
		return err
	}

	db.state.Set(service.Running, "")
	db.Logger.Info("Startup successful")
	return nil
}

// Stop shuts down the database service. If an error occurred during the shutdown,
// an error is returned.
// If the service is not running, this function does nothing.
func (db *Db) Stop(ctx context.Context) error {
	db.Logger.Info("Shutting down...")
	if code, _ := db.state.Get(); code != service.Running {
		db.Logger.Info("Service is already offline")
		return nil
	}
	db.state.Set(service.ShuttingDown, "")

	err := db.engine.Close()
	if err != nil {
		db.state.Set(service.Error, err.Error())
		db.Logger.Infof("Error while closing the database: %s", err)
		return err
	}
	db.state.Set(service.Offline, "")
	db.Logger.Info("Shutdown complete")
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
func (db *Db) State() *service.State {
	_ = ping(&db.state, db.engine, db.Logger)
	return &db.state
}

// Get retrieves one record from the database and fills the bean with it. Non-empty
// fields are used as conditions.
func (db *Db) Get(bean interface{}) error {
	db.Logger.Debugf("Get requested with %s", bean)

	if s, _ := db.state.Get(); s != service.Running {
		return ErrServiceUnavailable
	}
	if bean == nil {
		return ErrNilRecord
	}

	has, err := db.engine.Get(bean)
	if err != nil {
		if err := ping(&db.state, db.engine, db.Logger); err != nil {
			return err
		}

		return err
	}
	if !has {
		return ErrNotFound
	}
	return nil
}

// Select retrieves multiple records from the database using the given filters
// and fills the bean with it. The bean should be of type []Struct or []*Struct,
// and it should be empty.
func (db *Db) Select(bean interface{}, filters *Filters) error {
	db.Logger.Debugf("Select requested with %s", bean)
	if s, _ := db.state.Get(); s != service.Running {
		return ErrServiceUnavailable
	}
	if bean == nil {
		return ErrNilRecord
	}
	var query xorm.Interface = db.engine
	if filters != nil {
		query = query.Where(filters.Conditions).Limit(filters.Limit, filters.Offset).
			OrderBy(filters.Order)

	}

	if err := query.Find(bean); err != nil {
		if err := ping(&db.state, db.engine, db.Logger); err != nil {
			return err
		}
		return err
	}
	return nil
}

// Create inserts the given bean in the database. If the struct cannot be inserted,
// the function returns an error.
func (db *Db) Create(bean interface{}) error {
	db.Logger.Debugf("Create requested with %s", bean)
	if s, _ := db.state.Get(); s != service.Running {
		return ErrServiceUnavailable
	}
	if bean == nil {
		return ErrNilRecord
	}

	if _, err := db.engine.InsertOne(bean); err != nil {
		if err := ping(&db.state, db.engine, db.Logger); err != nil {
			return err
		}
		return err
	}
	return nil
}

// Update updates the given bean in the database. If the struct cannot be updated,
// the function returns an error.
func (db *Db) Update(before, after interface{}) error {
	db.Logger.Debugf("Update requested with %s", after)
	if s, _ := db.state.Get(); s != service.Running {
		return ErrServiceUnavailable
	}
	if before == nil || after == nil {
		return ErrNilRecord
	}

	if _, err := db.engine.Update(after, before); err != nil {
		if err := ping(&db.state, db.engine, db.Logger); err != nil {
			return err
		}
		return err
	}
	return nil
}

// Delete deletes the given bean from the database. If the record cannot be deleted,
// an error is returned.
func (db *Db) Delete(bean interface{}) error {
	db.Logger.Debugf("Delete requested with %s", bean)
	if s, _ := db.state.Get(); s != service.Running {
		return ErrServiceUnavailable
	}
	if bean == nil {
		return ErrNilRecord
	}

	if _, err := db.engine.Delete(bean); err != nil {
		if err := ping(&db.state, db.engine, db.Logger); err != nil {
			return err
		}
		return err
	}
	return nil
}

// Exists checks if the given record exists in the database. If the database
// cannot be queried, an error is returned.
func (db *Db) Exists(bean interface{}) (bool, error) {
	db.Logger.Debugf("Exists requested with %s", bean)
	if s, _ := db.state.Get(); s != service.Running {
		return false, ErrServiceUnavailable
	}
	if bean == nil {
		return false, ErrNilRecord
	}

	has, err := db.engine.Exist(bean)
	if err != nil {
		if err := ping(&db.state, db.engine, db.Logger); err != nil {
			return false, err
		}
		return false, err
	}
	return has, nil
}

// Execute executes the given SQL command. The command can be a raw string with
// arguments, or an xorm.Builder struct.
func (db *Db) Execute(sqlorArgs ...interface{}) error {
	db.Logger.Debugf("Execute requested with %s", sqlorArgs)
	if s, _ := db.state.Get(); s != service.Running {
		return ErrServiceUnavailable
	}

	if _, err := db.engine.Exec(sqlorArgs...); err != nil {
		if err := ping(&db.state, db.engine, db.Logger); err != nil {
			return err
		}
		return err
	}
	return nil
}

// Query executes the given SQL query and returns the result. The query can be
// a raw string with arguments, or an xorm.Builder struct.
func (db *Db) Query(sqlorArgs ...interface{}) ([]map[string]interface{}, error) {
	db.Logger.Debugf("Query requested with %s", sqlorArgs)
	if s, _ := db.state.Get(); s != service.Running {
		return nil, ErrServiceUnavailable
	}

	res, err := db.engine.QueryInterface(sqlorArgs...)
	if err != nil {
		if err := ping(&db.state, db.engine, db.Logger); err != nil {
			return nil, err
		}
		return nil, err
	}
	return res, nil
}

// BeginTransaction returns a new session on which a database transaction can
// be performed.
func (db *Db) BeginTransaction() (*Session, error) {
	if s, _ := db.state.Get(); s != service.Running {
		return nil, ErrServiceUnavailable
	}

	s := &Session{
		session: db.engine.NewSession(),
		logger:  db.Logger,
		state:   &db.state,
	}
	if err := s.session.Begin(); err != nil {
		if err := ping(&db.state, db.engine, db.Logger); err != nil {
			return nil, err
		}
		return nil, err
	}
	db.Logger.Debug("Transaction started")
	return s, nil
}

// Session is a struct used to perform transactions on the database. A session
// can be created with the `BeginTransaction` method. Once the transaction is
// complete, it can be committed using `Commit`, it can also be canceled using
// the `Rollback` function.
type Session struct {
	session *xorm.Session
	logger  *log.Logger
	state   *service.State
}

// Get adds a 'get' query to the transaction. If the query cannot be executed,
// an error is returned.
func (s *Session) Get(bean interface{}) error {
	s.logger.Debugf("Transaction 'Get' with %s", bean)
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
	s.logger.Debugf("Transaction 'Select' with %s", bean)
	if s, _ := s.state.Get(); s != service.Running {
		return ErrServiceUnavailable
	}
	if bean == nil {
		return ErrNilRecord
	}
	var query xorm.Interface = s.session
	if filters != nil {
		query = query.Where(filters.Conditions).Limit(filters.Limit, filters.Offset).
			OrderBy(filters.Order)
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
	s.logger.Debugf("Transaction 'Create' with %s", bean)
	if s, _ := s.state.Get(); s != service.Running {
		return ErrServiceUnavailable
	}
	if bean == nil {
		return ErrNilRecord
	}

	if _, err := s.session.InsertOne(bean); err != nil {
		if err := ping(s.state, s.session, s.logger); err != nil {
			return err
		}
		return err
	}
	return nil
}

// Update adds an 'update' query to the transaction. If the query cannot be executed,
// an error is returned.
func (s *Session) Update(before, after interface{}) error {
	s.logger.Debugf("Transaction 'Update' with %s", after)
	if s, _ := s.state.Get(); s != service.Running {
		return ErrServiceUnavailable
	}
	if before == nil || after == nil {
		return ErrNilRecord
	}

	if _, err := s.session.Update(after, before); err != nil {
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
	s.logger.Debugf("Transaction 'Delete' with %s", bean)
	if s, _ := s.state.Get(); s != service.Running {
		return ErrServiceUnavailable
	}
	if bean == nil {
		return ErrNilRecord
	}

	if _, err := s.session.Delete(bean); err != nil {
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
	s.logger.Debugf("Transaction 'Exists' with %s", bean)
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
	s.logger.Debugf("Transaction 'Execute' with %s", sqlorArgs)
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
	s.logger.Debugf("Transaction 'Execute' with %s", sqlorArgs)
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
	s.logger.Info("Rolling back changes")
	s.session.Close()
}

// Commit commits all the transactions pending operations to the database. If
// the commit fails, the changes are dropped, and an error is returned. After
// this function is returned, the session is closed and no more transactions can
// be performed using this instance.
func (s *Session) Commit() error {
	s.logger.Info("Committing transaction")
	defer s.session.Close()
	if s, _ := s.state.Get(); s != service.Running {
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
