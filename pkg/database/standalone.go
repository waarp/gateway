package database

import (
	"runtime/debug"
	"sync"
	"time"

	"code.waarp.fr/lib/log"
	"xorm.io/xorm"
	"xorm.io/xorm/core"
)

// Standalone is a struct used to execute standalone commands on the database.
type Standalone struct {
	engine   *xorm.Engine
	logger   *log.Logger
	sessions sync.Map
}

func (s *Standalone) newSession() *Session {
	return &Session{
		id:      time.Now().UnixNano(),
		session: s.engine.NewSession(),
		logger:  s.logger,
	}
}

// If a transaction takes more than this amount of time, print a warning message
// with a stack trace (for debugging purposes), because transactions should not
// take that long.
const warnDuration = 10 * time.Second

// TransactionFunc is the type representing a function meant to be executed inside
// a transaction using the Standalone.Transaction method.
type TransactionFunc func(*Session) error

// Transaction executes all the commands in the given function as a transaction.
// The transaction will be then be roll-backed or committed, depending on whether
// the function returned an error or not.
func (s *Standalone) Transaction(fun TransactionFunc) error {
	ses := s.newSession()

	if err := ses.session.Begin(); err != nil {
		s.logger.Error("Failed to start transaction: %s", err)

		return NewInternalError(err)
	}

	s.sessions.Store(ses.id, ses)
	defer s.sessions.Delete(ses.id)

	done := make(chan bool)

	defer func() {
		if err := ses.session.Close(); err != nil {
			s.logger.Warning("an error occurred while closing the session: %v", err)
		}

		close(done)
	}()

	stack := debug.Stack()

	go func() {
		timer := time.NewTimer(warnDuration)
		defer timer.Stop()

		select {
		case <-timer.C:
			s.logger.Warning("transaction is taking an unusually long time, "+
				"printing stack for debugging purposes:\n%s", stack)
		case <-done:
		}
	}()

	if err := fun(ses); err != nil {
		s.logger.Trace("Transaction failed, changes have been rolled back")

		if rbErr := ses.session.Rollback(); rbErr != nil {
			s.logger.Warning("an error occurred while rolling back the transaction: %v", rbErr)
		}

		return err
	}

	if err := ses.session.Commit(); err != nil {
		s.logger.Error("Failed to commit changes: %s", err)

		return NewInternalError(err)
	}

	return nil
}

func (s *Standalone) getUnderlying() xorm.Interface {
	return s.engine
}

// GetLogger returns the database logger instance.
func (s *Standalone) GetLogger() *log.Logger {
	return s.logger
}

// Iterate starts building a SQL 'SELECT' query to retrieve entries of the given
// model from the database. The request can be narrowed using the IterateQuery
// methods.
//
// The request can then be executed using the IterateQuery.Run method. The
// selected entries will be returned inside an Iterator instance.
func (s *Standalone) Iterate(bean IterateBean) *IterateQuery {
	return &IterateQuery{db: s, bean: bean}
}

// Select starts building a SQL 'SELECT' query to retrieve entries of the given
// model from the database. The request can be narrowed using the SelectQuery
// methods.
//
// The request can then be executed using the SelectQuery.Run method. The
// selected entries will be returned inside the SelectBean parameter.
func (s *Standalone) Select(bean SelectBean) *SelectQuery {
	return &SelectQuery{db: s, bean: bean}
}

// Get starts building a SQL 'SELECT' query to retrieve a single entry of
// the given model from the database. As a consequence, the function thus
// requires an SQL string with arguments to filter the said entry (the syntax is
// similar to the IterateQuery.Where method).
//
// The request can then be executed using the GetQuery.Run method. The bean
// parameter will be filled with the values retrieved from the database.
func (s *Standalone) Get(bean GetBean, where string, args ...interface{}) *GetQuery {
	return &GetQuery{db: s, bean: bean, conds: []*condition{{sql: where, args: args}}}
}

// Count starts building a SQL 'SELECT COUNT' query to count specific entries
// of the given model from the database. The request can be narrowed using
// the CountQuery.Where method.
//
// The request can then be executed using the IterateQuery.Run method. The
// selected entries will be returned inside an Iterator instance.
func (s *Standalone) Count(bean IterateBean) *CountQuery {
	return &CountQuery{db: s, bean: bean}
}

// Insert starts building a SQL 'INSERT' query to add the given model entry
// to the database.
//
// The request can then be executed using the InsertQuery.Run method.
func (s *Standalone) Insert(bean InsertBean) *InsertQuery {
	return &InsertQuery{db: s, bean: bean}
}

// Update starts building a SQL 'UPDATE' query to update single entry in
// the database, using the entry's ID as parameter. The request fails with
// an error if the entry does not exist.
//
// The request can then be executed using the UpdateQuery.Run method.
func (s *Standalone) Update(bean UpdateBean) *UpdateQuery {
	return &UpdateQuery{db: s, bean: bean}
}

// Delete starts building a SQL 'DELETE' query to delete a single entry of
// the given model from the database, using the entry's ID as parameter.
//
// The request can then be executed using the DeleteQuery.Run method.
func (s *Standalone) Delete(bean DeleteBean) *DeleteQuery {
	return &DeleteQuery{db: s, bean: bean}
}

// DeleteAll starts building a SQL 'DELETE' query to delete entries of the
// given model from the database. The request can be narrowed using the
// DeleteAllQuery.Where method.
//
// Be aware, since DeleteAll deletes multiple entries with only one SQL
// command, the model's `BeforeDelete` function will not be called when using
// this method. Thus, DeleteAll should exclusively be used on models with
// no DeletionHook.
//
// The request can then be executed using the DeleteAllQuery.Run method.
func (s *Standalone) DeleteAll(bean DeleteAllBean) *DeleteAllQuery {
	return &DeleteAllQuery{db: s, bean: bean}
}

// Exec executes the given custom SQL query, and returns any error encountered.
// The query uses the '?' character as a placeholder for arguments.
//
// Be aware that, since this method bypasses the data models, all the models'
// hooks will be skipped. Thus, this method should be used with extreme caution.
func (s *Standalone) Exec(query string, args ...interface{}) error {
	return exec(s.engine.NewSession(), s.logger, query, args...)
}

// QueryRow returns a single row from the database, which can then be scanned.
//
// Be aware that, since this method bypasses the data models, all the models'
// hooks will be skipped. Thus, this method should be used with caution.
func (s *Standalone) QueryRow(sql string, args ...any) *core.Row {
	return s.engine.DB().QueryRow(sql, args...)
}
