package database

import (
	"database/sql"
	"fmt"

	"code.waarp.fr/lib/log"
	"xorm.io/xorm"
	"xorm.io/xorm/schemas"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

// Session is a struct used to perform transactions on the database. A session
// can be performed using the Standalone.Transaction method.
type Session struct {
	id      int64
	db      *DB
	session *xorm.Session
	logger  *log.Logger
}

func (s *Session) getUnderlying() xorm.Interface {
	return s.session
}
func (s *Session) AsDB() *DB { return s.db }

// GetLogger returns the database logger instance.
func (s *Session) GetLogger() *log.Logger {
	return s.logger
}

func (s *Session) Transaction(fun TransactionFunc) error { return fun(s) }

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
	return &SelectQuery{db: s, bean: bean, forUpd: false}
}

// SelectForUpdate starts building a SQL 'SELECT FOR UPDATE' query to retrieve
// entries of the given model from the database. The request can be narrowed
// using the SelectQuery methods.
//
// The request can then be executed using the SelectQuery.Run method. The
// selected entries will be returned inside the SelectBean parameter.
func (s *Session) SelectForUpdate(bean SelectBean) *SelectQuery {
	return &SelectQuery{db: s, bean: bean, forUpd: true}
}

// Get starts building a SQL 'SELECT' query to retrieve a single entry of
// the given model from the database. The function also requires an SQL
// string with arguments to filter the result (similarly to the
// IterateQuery.Where method).
//
// The request can then be executed using the GetQuery.Run method. The bean
// parameter will be filled with the values retrieved from the database.
func (s *Session) Get(bean GetBean, where string, args ...any) *GetQuery {
	return &GetQuery{db: s, bean: bean, conds: []*condition{{sql: where, args: args}}}
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

// Delete starts building a SQL 'DELETE' query to delete a single entry of
// the given model from the database, using the entry's ID as parameter.
//
// The request can then be executed using the DeleteQuery.Run method.
func (s *Session) Delete(bean DeleteBean) *DeleteQuery {
	return &DeleteQuery{db: s, bean: bean}
}

// DeleteAll starts building an SQL 'DELETE' query to delete entries of the
// given model from the database. The request can be narrowed using the
// DeleteAllQuery.Where method.
//
// Be aware, since DeleteAll deletes multiple entries with only one SQL
// command, the model's `BeforeDelete` function will not be called when using
// this method. Thus DeleteAll should exclusively be used on models with no
// DeletionHook.
//
// The request can then be executed using the DeleteAllQuery.Run method.
func (s *Session) DeleteAll(bean DeleteAllBean) *DeleteAllQuery {
	return &DeleteAllQuery{db: s, bean: bean}
}

// Exec executes the given custom SQL query, and returns any error encountered.
// The query uses the '?' character as a placeholder for arguments.
//
// Be aware that, since this method bypasses the data models, all the models'
// hooks will be skipped. Thus, this method should be used with extreme caution.
func (s *Session) Exec(query string, args ...any) error {
	return exec(s.session, s.logger, query, args...)
}

// ResetIncrement resets the auto-increment on the given model's ID primary key.
// The auto-increment can only be reset if the table is empty.
func (s *Session) ResetIncrement(bean IterateBean) error {
	if n, err := s.session.NoAutoCondition().Count(bean); err != nil {
		s.logger.Errorf("Failed to query table %q: %v", bean.TableName(), err)

		return NewInternalError(err)
	} else if n != 0 {
		return NewValidationErrorf(
			"cannot reset the increment on table %q while there are still rows in it",
			bean.TableName())
	}

	var err error

	switch dbType := s.session.Engine().Dialect().URI().DBType; dbType {
	case schemas.SQLITE:
		_, err = s.session.Exec("DELETE FROM sqlite_sequence WHERE name=?", bean.TableName())
	case schemas.POSTGRES:
		_, err = s.session.Exec("TRUNCATE " + bean.TableName() + " RESTART IDENTITY CASCADE")
	case schemas.MYSQL:
		_, err = s.session.Exec("ALTER TABLE " + bean.TableName() + " AUTO_INCREMENT = 1")
	default:
		s.logger.Errorf("%s databases do not support resetting an auto-increment",
			conf.GlobalConfig.Database.Type)

		return &InternalError{
			msg:   fmt.Sprintf("unsupported database: %s", dbType),
			cause: errUnsupportedDB,
		}
	}

	if err != nil {
		s.logger.Errorf("Failed to reset the auto-increment on table %q: %v",
			bean.TableName(), err)

		return NewInternalError(err)
	}

	return nil
}

// AdvanceIncrement advances the auto-increment on the given model's ID primary
// key to the given value.
func (s *Session) AdvanceIncrement(bean Table, value int64) error {
	var maxID sql.NullInt64

	row := s.session.Tx().QueryRow("SELECT MAX(id) AS maxID FROM " + bean.TableName())

	if err := row.Scan(&maxID); err != nil {
		s.logger.Errorf("Failed to query table %q: %v", bean.TableName(), err)

		return NewInternalError(err)
	}

	if maxID.Valid && maxID.Int64 > value {
		return NewValidationErrorf("cannot advance the auto-increment of table "+
			"%q to a value (%d) lower than the current max value (%d) of the column",
			bean.TableName(), value, maxID.Int64)
	}

	var err error

	switch dbType := s.session.Engine().Dialect().URI().DBType; dbType {
	case schemas.SQLITE:
		if _, delErr := s.session.Exec("DELETE FROM sqlite_sequence WHERE name=?",
			bean.TableName()); delErr != nil {
			s.logger.Errorf("Failed to delete the SQLite auto-increment on table %q: %v",
				bean.TableName(), err)

			return NewInternalError(err)
		}

		_, err = s.session.Exec("INSERT INTO sqlite_sequence(seq,name) VALUES (?,?)",
			value, bean.TableName())
	case schemas.POSTGRES:
		_, err = s.session.Exec("SELECT setval(pg_get_serial_sequence(?, 'id'), ?)",
			bean.TableName(), value)
	case schemas.MYSQL:
		_, err = s.session.Exec("ALTER TABLE " + bean.TableName() + " AUTO_INCREMENT = " +
			utils.FormatInt(value+1))
	default:
		s.logger.Errorf("%s databases do not support resetting an auto-increment",
			conf.GlobalConfig.Database.Type)

		return &InternalError{
			msg:   fmt.Sprintf("unsupported database: %s", dbType),
			cause: errUnsupportedDB,
		}
	}

	if err != nil {
		s.logger.Errorf("Failed to advance the auto-increment on table %q: %v",
			bean.TableName(), err)

		return NewInternalError(err)
	}

	return nil
}

// QueryRow returns a single row from the database, which can then be scanned.
//
// Be aware that, since this method bypasses the data models, all the models'
// hooks will be skipped. Thus, this method should be used with caution.
func (s *Session) QueryRow(query string, args ...any) *sql.Row {
	return s.session.DB().DB.QueryRow(query, args...)
}
