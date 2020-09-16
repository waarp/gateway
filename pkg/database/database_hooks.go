package database

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
	ElemName() string
}

// identifier is an interface which adds a function which returns the entry's
// ID number. Models must implement this interface in order to be updated.
type identifier interface {
	GetID() uint64
}

type entry interface {
	tableName
	identifier
}
