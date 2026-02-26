package database

import (
	"context"
	"database/sql"
	"time"

	"xorm.io/xorm"

	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
)

func (db *DB) newSession() *Session {
	return &Session{
		id:      time.Now().UnixNano(),
		db:      db,
		session: db.engine.NewSession(),
		logger:  db.logger,
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
func (db *DB) Transaction(fun TransactionFunc) error {
	ses := db.newSession()

	if err := ses.session.Begin(); err != nil {
		db.logger.Errorf("Failed to start transaction: %v", err)

		return NewInternalError(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), warnDuration)
	defer cancel()

	ses.session.Context(ctx)

	db.sessions.Store(ses.id, ses)
	defer db.sessions.Delete(ses.id)

	done := make(chan bool)

	defer func() {
		if err := ses.session.Close(); err != nil {
			db.logger.Warningf("an error occurred while closing the session: %v", err)
		}

		close(done)
	}()

	if err := fun(ses); err != nil {
		db.logger.Trace("Transaction failed, changes have been rolled back")

		if rbErr := ses.session.Rollback(); rbErr != nil {
			db.logger.Warningf("an error occurred while rolling back the transaction: %v", rbErr)
		}

		return err
	}

	if err := ses.session.Commit(); err != nil {
		db.logger.Errorf("Failed to commit changes: %v", err)

		return NewInternalError(err)
	}

	return nil
}

func (db *DB) getUnderlying() xorm.Interface {
	return db.engine
}

// GetLogger returns the database logger instance.
func (db *DB) GetLogger() *log.Logger {
	return db.logger
}

func (db *DB) AsDB() *DB { return db }

// Iterate starts building a SQL 'SELECT' query to retrieve entries of the given
// model from the database. The request can be narrowed using the IterateQuery
// methods.
//
// The request can then be executed using the IterateQuery.Run method. The
// selected entries will be returned inside an Iterator instance.
func (db *DB) Iterate(bean IterateBean) *IterateQuery {
	return &IterateQuery{db: db, bean: bean}
}

// Select starts building a SQL 'SELECT' query to retrieve entries of the given
// model from the database. The request can be narrowed using the SelectQuery
// methods.
//
// The request can then be executed using the SelectQuery.Run method. The
// selected entries will be returned inside the SelectBean parameter.
func (db *DB) Select(bean SelectBean) *SelectQuery {
	return &SelectQuery{db: db, bean: bean}
}

// Get starts building a SQL 'SELECT' query to retrieve a single entry of
// the given model from the database. As a consequence, the function thus
// requires an SQL string with arguments to filter the said entry (the syntax is
// similar to the IterateQuery.Where method).
//
// The request can then be executed using the GetQuery.Run method. The bean
// parameter will be filled with the values retrieved from the database.
func (db *DB) Get(bean GetBean, where string, args ...any) *GetQuery {
	return &GetQuery{db: db, bean: bean, conds: []*condition{{sql: where, args: args}}}
}

// Count starts building a SQL 'SELECT COUNT' query to count specific entries
// of the given model from the database. The request can be narrowed using
// the CountQuery.Where method.
//
// The request can then be executed using the CountQuery.Run method.
func (db *DB) Count(bean IterateBean) *CountQuery {
	return &CountQuery{db: db, bean: bean}
}

// Insert starts building a SQL 'INSERT' query to add the given model entry
// to the database.
//
// The request can then be executed using the InsertQuery.Run method.
func (db *DB) Insert(bean InsertBean) *InsertQuery {
	return &InsertQuery{db: db, bean: bean}
}

// Update starts building a SQL 'UPDATE' query to update single entry in
// the database, using the entry's ID as parameter. The request fails with
// an error if the entry does not exist.
//
// The request can then be executed using the UpdateQuery.Run method.
func (db *DB) Update(bean UpdateBean) *UpdateQuery {
	return &UpdateQuery{db: db, bean: bean}
}

// Delete starts building a SQL 'DELETE' query to delete a single entry of
// the given model from the database, using the entry's ID as parameter.
//
// The request can then be executed using the DeleteQuery.Run method.
func (db *DB) Delete(bean DeleteBean) *DeleteQuery {
	return &DeleteQuery{db: db, bean: bean}
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
func (db *DB) DeleteAll(bean DeleteAllBean) *DeleteAllQuery {
	return &DeleteAllQuery{db: db, bean: bean}
}

// Exec executes the given custom SQL query, and returns any error encountered.
// The query uses the '?' character as a placeholder for arguments.
//
// Be aware that, since this method bypasses the data models, all the models'
// hooks will be skipped. Thus, this method should be used with extreme caution.
func (db *DB) Exec(query string, args ...any) error {
	return exec(db.engine.NewSession(), db.logger, query, args...)
}

// QueryRow returns a single row from the database, which can then be scanned.
//
// Be aware that, since this method bypasses the data models, all the models'
// hooks will be skipped. Thus, this method should be used with caution.
func (db *DB) QueryRow(query string, args ...any) *sql.Row {
	return db.engine.DB().DB.QueryRow(query, args...)
}
