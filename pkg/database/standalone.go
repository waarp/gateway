package database

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"xorm.io/xorm"
)

// Standalone is a struct used to execute standalone commands on the database.
type Standalone struct {
	engine *xorm.Engine
	logger *log.Logger
}

func (s *Standalone) newSession() *Session {
	return &Session{
		session: s.engine.NewSession(),
		logger:  s.logger,
	}
}

// Transaction executes all the commands in the given function as a transaction.
// The transaction will be then be roll-backed or committed, depending whether
// the function returned an error or not.
func (s *Standalone) Transaction(f func(*Session) Error) Error {
	return s.transaction(false, f)
}

// WriteTransaction executes all the commands in the given function as a transaction.
// The difference between this function and Transaction is that, on an SQLite
// database, this function will open an EXCLUSIVE transaction instead of a normal
// one, preventing other connections from opening new transactions while this one
// is active.
//
// Generally, if your transaction function calls Session.SelectForUpdate, you
// should execute the transaction using this method instead of Transaction.
func (s *Standalone) WriteTransaction(f func(*Session) Error) Error {
	return s.transaction(true, f)
}

func (s *Standalone) transaction(isWrite bool, f func(*Session) Error) Error {
	ses := s.newSession()

	if err := ses.session.Begin(); err != nil {
		s.logger.Errorf("Failed to start transaction: %s", err)
		return NewInternalError(err)
	}
	if isWrite && conf.GlobalConfig.Database.Type == SQLite {
		if _, err := ses.session.Exec("ROLLBACK; BEGIN IMMEDIATE"); err != nil {
			s.logger.Errorf("Failed to start transaction: %s", err)
			return &InternalError{msg: "failed to start transaction", cause: err}
		}
	}
	defer func() { _ = ses.session.Close() }()

	s.logger.Debug("[SQL] Beginning transaction")
	if err := f(ses); err != nil {
		s.logger.Error("Transaction failed, changes have been rolled back")
		_ = ses.session.Rollback()
		return err
	}
	if err := ses.session.Commit(); err != nil {
		s.logger.Errorf("Failed to commit changes: %s", err)
		return NewInternalError(err)
	}
	s.logger.Debug("[SQL] Transaction succeeded, changes have been committed")
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
	return &GetQuery{db: s, bean: bean, sql: where, args: args}
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

// UpdateAll starts building an SQL 'UPDATE' query to update multiple entries
// in the database. The columns to update, and their values are specified
// using the UpdVals parameter. The entries to update can be filtered using
// the sql & args parameters, with a syntax similar to the IterateQuery.Where
// method.
//
// Be aware that, since this method updates multiple rows at once, the entries'
// WriteHook will NOT be executed. Thus, this method should be used with
// extreme caution.
//
// The request can then be executed using the UpdateAllQuery.Run method.
func (s *Standalone) UpdateAll(bean UpdateAllBean, vals UpdVals, sql string,
	args ...interface{}) *UpdateAllQuery {
	return &UpdateAllQuery{db: s, bean: bean, vals: vals, conds: sql, args: args}
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
// this method. Thus DeleteAll should exclusively be used on models with
// with no DeletionHook.
//
// The request can then be executed using the DeleteAllQuery.Run method.
func (s *Standalone) DeleteAll(bean DeleteAllBean) *DeleteAllQuery {
	return &DeleteAllQuery{db: s, bean: bean}
}
