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

// insertHook is an interface which adds a function called just before inserting
// a new entry in the database.
type insertHook interface {
	BeforeInsert(Accessor) error
}

// updateHook is an interface which adds a function called just before updating
// an entry in the database.
type updateHook interface {
	BeforeUpdate(Accessor, uint64) error
}

// deleteHook is an interface which adds a function called just before deleting
// an entry from the database.
type deleteHook interface {
	BeforeDelete(Accessor) error
}
