package database

// insertHook is an interface which adds a function called just before inserting
// a new entry in the database.
type insertHook interface {
	BeforeInsert(Accessor) error
}

// updateHook is an interface which adds a function called just before updating
// an entry in the database.
type updateHook interface {
	BeforeUpdate(Accessor) error
}

// deleteHook is an interface which adds a function called just before deleting
// an entry from the database.
type deleteHook interface {
	BeforeDelete(Accessor) error
}
