package model

import "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"

// Validator adds a 'Validate' function which checks if an entry can be inserted
// in database.
type Validator interface {
	// Validate checks if the struct can be inserted in the database
	Validate(db *database.Db, isInsert bool) error
}

// ErrInvalid is returned by 'Validate' if the entry is invalid.
type ErrInvalid struct {
	msg string
}

func (e ErrInvalid) Error() string {
	return e.msg
}
