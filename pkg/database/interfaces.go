package database

import (
	"database/sql"
)

// Deprecated:
// validator is an interface which adds a function called just before inserting
// a new entry in the database.
type validator interface {
	Validate(Accessor) error
}

// Deprecated:
// deleteHook is an interface which adds a function called just before deleting
// an entry from the database.
type deleteHook interface {
	BeforeDelete(Accessor) error
}

// Deprecated:
// tableName is the interface that models MUST implement in order to have a
// corresponding table in the database.
type tableName interface {
	TableName() string
	//Deprecated:
	ElemName() string
}

// identifier is an interface which adds a function which returns the entry's
// ID number. Models must implement this interface in order to be updated.
type identifier interface {
	GetID() uint64
}

// Deprecated:
type entry interface {
	tableName
	identifier
}

type command interface {
	exec(*Session) (sql.Result, error)
}

type query interface {
	query(*Session) (Iterator, error)
}

// deletionHook is an interface which adds a function which will be run before
// deleting an entry.
type deletionHook interface {
	BeforeDelete(*Session) error
}

// writeHook is an interface which adds a function which will be run before
// inserting or updating an entry.
type writeHook interface {
	BeforeWrite(*Session) error
}

// table is an interface which adds a function which returns the name of
// the table to which the element belongs.
type table interface {
	Table() string
}

// appellation is an interface which adds a function which returns the usage name
// of a table the element. This is used for logging and error reporting.
type appellation interface {
	Appellation() string
}

// Iterator is the interface returned by the DB.Query2 function. It allows to
// iterate over the rows selected by the statement, with the Next and Scan
// functions. Scan allows to directly write the content of the row into an
// instance of the corresponding model.
type Iterator interface {
	Next() bool
	Scan(interface{}) error
	Close() error
}

type dbResult struct {
	insertID, affected int64
}

func (d *dbResult) LastInsertId() (int64, error) { return d.insertID, nil }
func (d *dbResult) RowsAffected() (int64, error) { return d.affected, nil }

type saIterator struct {
	Iterator
	closeSes func()
}

func (s *saIterator) Close() error {
	defer s.closeSes()
	return s.Iterator.Close()
}
