package database

import (
	"database/sql"
	"reflect"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
	"github.com/go-xorm/builder"
	"github.com/go-xorm/xorm"
)

// Session is a struct used to perform transactions on the database. A session
// can be created with the `BeginTransaction` method. Once the transaction is
// complete, it can be committed using `Commit`, it can also be canceled using
// the `Rollback` function.
type Session struct {
	session *xorm.Session
	logger  *log.Logger
	state   *service.State
	cancel  func()
}

func (db *DB) newSession() *Session {
	return &Session{
		session: db.engine.NewSession(),
		logger:  db.logger,
		state:   &db.state,
	}
}

func (s *Session) commit() error {
	defer s.session.Close()
	return s.session.Commit()
}

func (s *Session) rollback() {
	s.session.Close()
}

func (s *Session) logSQL() {
	msg, err := builder.ConvertToBoundSQL(s.session.LastSQL())
	if err != nil {
		s.logger.Warningf("Failed to log SQL command: %s", err)
	} else {
		s.logger.Debug(msg)
	}
}

// Exec executes the given SQL command. The command can be created using the
// various command builder functions available in this package, such as:
// - Insert
// - Update
// - Delete
func (s *Session) Exec(cmd command) (sql.Result, error) {
	defer s.logSQL()
	return cmd.exec(s)
}

// Query2 executes the given SQL query and returns the selected database rows
// as an Iterator. The query can be created using the various query builder
// functions available in this package, such as:
// - Select
func (s *Session) Query2(query query) (Iterator, error) {
	defer s.logSQL()
	return query.query(s)
}

// Count is a wrapper function which counts the number of rows returned by a
// 'SELECT' query without having to iterate on all the rows.
func (s *Session) Count(query *selectQuery) (int64, error) {
	defer s.logSQL()
	if query.cond != nil {
		s.session.Where(query.cond.convert())
	}
	bean := reflect.New(reflect.TypeOf(query.bean).Elem()).Interface().(selectBean)
	return s.session.Count(bean)
}

// Get adds a 'get' query to the transaction. If the query cannot be executed,
// an error is returned.
// Deprecated:
func (s *Session) Get(bean tableName) error {
	s.logger.Debugf("Transaction 'Get' with %#v", bean)
	if s, _ := s.state.Get(); s != service.Running {
		return ErrServiceUnavailable
	}
	if bean == nil {
		return ErrNilRecord
	}

	val := reflect.ValueOf(bean).Elem()
	if val.Type().Name() == "Rule" && !val.FieldByName("Name").IsZero() {
		s.session.UseBool("send")
	}

	if has, err := s.session.Get(bean); err != nil {
		if pErr := ping(s.state, s.session, s.logger); pErr != nil {
			return pErr
		}
		return NewInternalError(err, "failed to retrieve the requested entry")
	} else if !has {
		return newNotFoundError(bean.ElemName())
	}
	return nil
}

// Select adds a 'select' query to the transaction with the conditions given in
// the `filters` parameter.  If the query cannot be executed, an error is returned.
// Deprecated:
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
		if pErr := ping(s.state, s.session, s.logger); pErr != nil {
			return pErr
		}
		return NewInternalError(err, "failed to retrieve the requested entries")
	}
	return nil
}

func (s *Session) create(bean tableName) error {
	if hook, ok := bean.(validator); ok {
		if err := hook.Validate(s); err != nil {
			return err
		}
	}

	if _, err := s.session.InsertOne(bean); err != nil {
		return NewInternalError(err, "failed to insert entry")
	}

	return nil
}

// Create adds an 'insert' query to the transaction. If the query cannot be executed,
// an error is returned.
// Deprecated:
func (s *Session) Create(bean tableName) error {
	s.logger.Debugf("Transaction 'Create' with %#v", bean)
	if s, _ := s.state.Get(); s != service.Running {
		return ErrServiceUnavailable
	}
	if bean == nil {
		return ErrNilRecord
	}

	if err := s.create(bean); err != nil {
		if pErr := ping(s.state, s.session, s.logger); pErr != nil {
			return pErr
		}
		return err
	}
	return nil
}

func (s *Session) update(bean entry) error {
	if hook, ok := bean.(validator); ok {
		if err := hook.Validate(s); err != nil {
			return err
		}
	}

	if _, err := s.session.AllCols().ID(bean.GetID()).Update(bean); err != nil {
		e := NewInternalError(err, "failed to update %s", bean.ElemName())
		s.logger.Errorf("Failed to update %s: %s", bean.ElemName(), err)
		return e
	}

	return nil
}

// Update adds an 'update' query to the transaction. If the query cannot be executed,
// an error is returned.
// Deprecated:
func (s *Session) Update(bean entry) error {
	s.logger.Debugf("Transaction 'Update' with %#v", bean)
	if s, _ := s.state.Get(); s != service.Running {
		return ErrServiceUnavailable
	}
	if bean == nil {
		return ErrNilRecord
	}
	query := builder.Select("id").From(bean.TableName()).Where(builder.Eq{"id": bean.GetID()})
	if res, err := s.session.Query(query); err != nil {
		return NewInternalError(err, "failed to retrieve the %s to update", bean.ElemName())
	} else if len(res) == 0 {
		return newNotFoundError(bean.ElemName())
	}

	if err := s.update(bean); err != nil {
		if pErr := ping(s.state, s.session, s.logger); pErr != nil {
			return pErr
		}
		return err
	}
	return nil
}

func (s *Session) delete(bean tableName) error {
	if hook, ok := bean.(deleteHook); ok {
		if err := hook.BeforeDelete(s); err != nil {
			return err
		}
	}

	if _, err := s.session.Delete(bean); err != nil {
		return NewInternalError(err, "failed to delete %s", bean.ElemName())
	}

	return nil
}

// Delete adds an 'delete' query to the transaction. If the query cannot be executed,
// an error is returned.
// Deprecated:
func (s *Session) Delete(bean tableName) error {
	s.logger.Debugf("Transaction 'Delete' with %#v", bean)
	if s, _ := s.state.Get(); s != service.Running {
		return ErrServiceUnavailable
	}
	if bean == nil {
		return ErrNilRecord
	}
	if exist, err := s.session.Exist(bean); err != nil {
		return NewInternalError(err, "failed to retrieve the %s to delete", bean.ElemName())
	} else if !exist {
		return newNotFoundError(bean.ElemName())
	}

	if err := s.delete(bean); err != nil {
		if pErr := ping(s.state, s.session, s.logger); pErr != nil {
			return pErr
		}
		return err
	}
	return nil
}

// Execute adds a custom raw query to the transaction. If the query cannot be executed,
// an error is returned. If the command must return a result, use `Query` instead.
// Deprecated:
func (s *Session) Execute(sqlorArgs ...interface{}) error {
	s.logger.Debugf("Transaction 'Execute' with %#v", sqlorArgs)
	if s, _ := s.state.Get(); s != service.Running {
		return ErrServiceUnavailable
	}

	if _, err := s.session.Exec(sqlorArgs...); err != nil {
		if pErr := ping(s.state, s.session, s.logger); pErr != nil {
			return pErr
		}
		return NewInternalError(err, "failed to execute SQL command")
	}
	return nil
}

// Query adds a custom raw query to the transaction. If the query cannot be executed,
// an error is returned. The function returns a slice of map[string]interface{}
// which contains the result of the query.
// Deprecated:
func (s *Session) Query(sqlorArgs ...interface{}) ([]map[string]interface{}, error) {
	s.logger.Debugf("Transaction 'Execute' with %#v", sqlorArgs)
	if s, _ := s.state.Get(); s != service.Running {
		return nil, ErrServiceUnavailable
	}

	res, err := s.session.QueryInterface(sqlorArgs...)
	if err != nil {
		if pErr := ping(s.state, s.session, s.logger); pErr != nil {
			return nil, pErr
		}
		return nil, NewInternalError(err, "failed to execute SQL query")
	}
	return res, nil
}

// Rollback cancels the transaction, and rolls back any changes made to the
// database. When this function is called, the session is closed, which means
// it cannot be used to perform any more transactions.
// Deprecated:
func (s *Session) Rollback() {
	defer s.cancel()
	s.logger.Debug("Rolling back transaction changes")
	s.session.Close()
}

// Commit commits all the transactions pending operations to the database. If
// the commit fails, the changes are dropped, and an error is returned. After
// this function is returned, the session is closed and no more transactions can
// be performed using this instance.
// Deprecated:
func (s *Session) Commit() error {
	s.logger.Debug("Committing transaction")
	defer func() {
		s.session.Close()
		s.cancel()
	}()

	if st, _ := s.state.Get(); st != service.Running {
		return ErrServiceUnavailable
	}

	if err := s.session.Commit(); err != nil {
		s.logger.Errorf("Commit failed (%s), changes were not committed", err)
		if pErr := ping(s.state, s.session, s.logger); pErr != nil {
			return pErr
		}
		return NewInternalError(err, "failed to commit transaction")
	}
	return nil
}
