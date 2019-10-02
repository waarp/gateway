package database

import (
	"fmt"
)

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

// insertValidator is an interface which adds a function called to validate an
// entry before inserting it in the database.
type insertValidator interface {
	ValidateInsert(ses *Session) error
}

// updateValidator is an interface which adds a function called to validate an
// entry before updating it in the database.
type updateValidator interface {
	ValidateUpdate(ses *Session, id uint64) error
}
