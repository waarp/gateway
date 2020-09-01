package database

import "fmt"

// ErrInvalid represents an error stating that a database entry is not valid for
// insertion or update.
type ErrInvalid struct {
	Msg string
}

func (e ErrInvalid) Error() string {
	return e.Msg
}

// InvalidError returns a new `ErrInvalid` with the given formatted message.
func InvalidError(msg string, args ...interface{}) *ErrInvalid {
	return &ErrInvalid{Msg: fmt.Sprintf(msg, args...)}
}

// validator is an interface which adds a function called just before inserting
// a new entry in the database.
type validator interface {
	Validate(Accessor) error
}

// deleteHook is an interface which adds a function called just before deleting
// an entry from the database.
type deleteHook interface {
	BeforeDelete(Accessor) error
}

// tableName is the interface that models MUST implement in order to have a
// corresponding table in the database.
type tableName interface {
	TableName() string
}

// identifier is an interface which adds a function which returns the entry's
// ID number. Models must implement this interface in order to be updated.
type identifier interface {
	Id() uint64
}

type entry interface {
	tableName
	identifier
}
