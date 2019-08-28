package model

// Tables lists the schema of all database tables
var Tables = make([]interface{}, 0)

// Validator adds a 'Validate' function which checks if an entry can be inserted
// in database.
type Validator interface {
	// Validate checks if the struct can be inserted in the database
	Validate(func(interface{}) (bool, error)) error
}

// ErrInvalid is returned by 'Validate' if the entry is invalid.
type ErrInvalid struct {
	msg string
}

func (e ErrInvalid) Error() string {
	return e.msg
}
