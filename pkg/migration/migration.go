// Package migration contains the elements for writing and running database
// migration operations.
package migration

// Script represents a single database migration. It is recommended to keep
// migrations short, and not to run many operation in a migration.
type Script struct {
	// The SQL script to execute. Up applies the migration, Down undoes it.
	Up   func(db Dialect) error
	Down func(db Dialect) error
}

// Migration is a single entry of the Migrations list. It contains the migration script
// itself, a description of the migration and (optionally) a version tag. The
// version tag should be unique, and they should be declared in order.
type Migration struct {
	Description string
	Script      Script
	Version     string
}
