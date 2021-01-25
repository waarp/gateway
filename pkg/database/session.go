package database

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
	"xorm.io/xorm"
)

// Session is a struct used to perform transactions on the database. A session
// can be performed using the Standalone.Transaction method.
type Session struct {
	session *xorm.Session
	logger  *log.Logger
	state   *service.State
}

func (s *Session) getUnderlying() xorm.Interface {
	return s.session
}

// GetLogger returns the database logger instance.
func (s *Session) GetLogger() *log.Logger {
	return s.logger
}

// Iterate starts building a SQL 'SELECT' query to retrieve entries of the given
// model from the database. The request can be narrowed using the IterateQuery
// methods.
//
// The request can then be executed using the IterateQuery.Run method. The
// selected entries will be returned inside an Iterator instance.
func (s *Session) Iterate(bean IterateBean) *IterateQuery {
	return &IterateQuery{db: s, bean: bean}
}

// Select starts building a SQL 'SELECT' query to retrieve entries of the given
// model from the database. The request can be narrowed using the SelectQuery
// methods.
//
// The request can then be executed using the SelectQuery.Run method. The
// selected entries will be returned inside the SelectBean parameter.
func (s *Session) Select(bean SelectBean) *SelectQuery {
	return &SelectQuery{db: s, bean: bean}
}

// Get starts building a SQL 'SELECT' query to retrieve a single entry of
// the given model from the database. The function also requires an SQL
// string with arguments to filter the result (similarly to the
// IterateQuery.Where method).
//
// The request can then be executed using the GetQuery.Run method. The bean
// parameter will be filled with the values retrieved from the database.
func (s *Session) Get(bean GetBean, where string, args ...interface{}) *GetQuery {
	return &GetQuery{db: s, bean: bean, sql: where, args: args}
}

// Count starts building a SQL 'SELECT COUNT' query to count specific entries
// of the given model from the database. The request can be narrowed using
// the CountQuery.Where method.
//
// The request can then be executed using the IterateQuery.Run method. The
// selected entries will be returned inside an Iterator instance.
func (s *Session) Count(bean IterateBean) *CountQuery {
	return &CountQuery{db: s, bean: bean}
}

// Insert starts building a SQL 'INSERT' query to add the given model entry
// to the database.
//
// The request can then be executed using the InsertQuery.Run method.
func (s *Session) Insert(bean InsertBean) *InsertQuery {
	return &InsertQuery{db: s, bean: bean}
}

// Update starts building a SQL 'UPDATE' query to update single entry in
// the database, using the entry's ID as parameter. The request fails with
// an error if the entry does not exist.
//
// The request can then be executed using the UpdateQuery.Run method.
func (s *Session) Update(bean UpdateBean) *UpdateQuery {
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
func (s *Session) UpdateAll(bean UpdateAllBean, vals UpdVals, sql string,
	args ...interface{}) *UpdateAllQuery {
	return &UpdateAllQuery{db: s, bean: bean, vals: vals, conds: sql, args: args}
}

// Delete starts building a SQL 'DELETE' query to delete a single entry of
// the given model from the database, using the entry's ID as parameter.
//
// The request can then be executed using the DeleteQuery.Run method.
func (s *Session) Delete(bean DeleteBean) *DeleteQuery {
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
func (s *Session) DeleteAll(bean DeleteAllBean) *DeleteAllQuery {
	return &DeleteAllQuery{db: s, bean: bean}
}
