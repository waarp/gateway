// Package migration contains the elements for writing and running database
// migration operations.
package migration

// Migration represents a single database migration. It is recommended to keep
// migrations short, and not to run many operation in a migration.
type Migration struct {
	// A human description of what the migration does.
	Description string

	// The SQL script to execute. Up applies the migration, Down undoes it.
	Up   func(Dialect) error
	Down func(Dialect) error
}
