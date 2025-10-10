package database

import (
	"database/sql"
	"fmt"

	"code.waarp.fr/lib/log"
	"xorm.io/xorm"
)

// ReadAccess is the interface listing all the read operations possible on the
// database. The interface defines a query building function for each of these
// operations. These functions all take a model, which represents the target of
// the operation; and then return a query builder.
//
// Depending on the operation, these query builders might expose a few functions
// to make these operation more precise. Once the query is defined, it can be
// executed using the `Run` function.
type ReadAccess interface {
	getUnderlying() xorm.Interface
	AsDB() *DB

	// GetLogger returns the database logger instance.
	GetLogger() *log.Logger

	// Iterate starts building a SQL 'SELECT' query to retrieve entries of the given
	// model from the database. The request can be narrowed using the IterateQuery
	// methods.
	//
	// The request can then be executed using the IterateQuery.Run method. The
	// selected entries will be returned inside an Iterator instance.
	Iterate(obj IterateBean) *IterateQuery

	// Select starts building a SQL 'SELECT' query to retrieve entries of the given
	// model from the database. The request can be narrowed using the SelectQuery
	// methods.
	//
	// The request can then be executed using the SelectQuery.Run method. The
	// selected entries will be returned inside the SelectBean parameter.
	Select(obj SelectBean) *SelectQuery

	// Get starts building a SQL 'SELECT' query to retrieve a single entry of
	// the given model from the database. The function also requires an SQL
	// string with arguments to filter the result (similarly to the
	// IterateQuery.Where method).
	//
	// The request can then be executed using the GetQuery.Run method. The bean
	// parameter will be filled with the values retrieved from the database.
	Get(obj GetBean, where string, args ...any) *GetQuery

	// Count starts building a SQL 'SELECT COUNT' query to count specific entries
	// of the given model from the database. The request can be narrowed using
	// the CountQuery.Where method.
	//
	// The request can then be executed using the CountQuery.Run method.
	Count(obj IterateBean) *CountQuery

	QueryRow(query string, args ...any) *sql.Row
}

// Access is the interface listing all the write operations possible on the
// database. The interface defines a query building function for each of these
// operations. These functions all take a model, which represents the target of
// the operation; and then return a query builder.
//
// Depending on the operation, these query builders might expose a few functions
// to make these operation more precise. Once the query is defined, it can be
// executed using the `Run` function.
type Access interface {
	ReadAccess

	// Insert starts building a SQL 'INSERT' query to add the given model entry
	// to the database.
	//
	// The request can then be executed using the InsertQuery.Run method.
	Insert(obj InsertBean) *InsertQuery

	// Update starts building a SQL 'UPDATE' query to update single entry in
	// the database, using the entry's ID as parameter. The request fails with
	// an error if the entry does not exist.
	//
	// The request can then be executed using the UpdateQuery.Run method.
	Update(obj UpdateBean) *UpdateQuery

	// Delete starts building a SQL 'DELETE' query to delete a single entry of
	// the given model from the database, using the entry's ID as parameter.
	//
	// The request can then be executed using the DeleteQuery.Run method.
	Delete(obj DeleteBean) *DeleteQuery

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
	DeleteAll(obj DeleteAllBean) *DeleteAllQuery

	// Exec executes the given custom SQL query, and returns any error encountered.
	// The query uses the '?' character as a placeholder for arguments.
	//
	// Be aware that, since this method bypasses the data models, all the models'
	// hooks will be skipped. Thus, this method should be used with extreme caution.
	Exec(query string, args ...any) error

	// Transaction executes all the commands in the given function as a transaction.
	// The transaction will be then be roll-backed or committed, depending on whether
	// the function returned an error or not.
	Transaction(fun TransactionFunc) error
}

// DeletionHook is an interface which adds a function which will be run before
// deleting an entry.
type DeletionHook interface {
	BeforeDelete(db Access) error
}

// WriteHook is an interface which adds a function which will be run before
// inserting or updating an entry.
type WriteHook interface {
	BeforeWrite(db Access) error
}

// InsertCallback is an interface which adds a function which will be run after
// inserting a new entry.
type InsertCallback interface {
	AfterInsert(db Access) error
}

// UpdateCallback is an interface which adds a function which will be run after
// updating an entry.
type UpdateCallback interface {
	AfterUpdate(db Access) error
}

// ReadCallback is an interface which adds a function which will be run after an
// entry has been retrieved from the database.
type ReadCallback interface {
	AfterRead(db ReadAccess) error
}

// Table is the interface which adds the base methods that all database models
// must implement.
//
//nolint:iface //best keep generic table interface separate from bean interfaces
type Table interface {
	// TableName returns the name of the table (as defined in the database).
	TableName() string

	// Appellation returns the natural name used to designate a single entry.
	// This is mostly used for error reporting purposes.
	Appellation() string
}

// Identifier is an interface which adds a function which returns the entry's
// ID number. Models must implement this interface in order to be updated.
type Identifier interface {
	// GetID returns the entry's ID.
	GetID() int64
}

// Iterator is the object returned when sending a 'SELECT' query to the database.
// It allows to iterate over the rows selected by the statement, with the Next
// and Scan functions. Scan allows to directly write the content of the row into an
// instance of the corresponding model.
//
// Iterator instances MUST be closed once all the entries have been retrieved.
type Iterator struct {
	*xorm.Rows

	db     ReadAccess
	logger *log.Logger
}

// Scan parses the current line of the iterator, and fills the given model
// parameter with the parsed values. Returns an error if the line cannot be
// retrieved, or if the parsed line does not correspond to the given model.
func (i *Iterator) Scan(bean IterateBean) error {
	if err := i.Rows.Scan(bean); err != nil {
		return fmt.Errorf("cannot scan database row: %w", err)
	}

	if hook, ok := bean.(ReadCallback); ok {
		if err := hook.AfterRead(i.db); err != nil {
			i.logger.Errorf("%s entry GET callback failed: %v", bean.Appellation(), err)

			return fmt.Errorf("%s entry ITERATE callback failed: %w", bean.Appellation(), err)
		}
	}

	return nil
}

// Close closes the iterator, and releases the connection to the database.
func (i *Iterator) Close() {
	_ = i.Rows.Close() //nolint:errcheck // nothing to handle the error
}
