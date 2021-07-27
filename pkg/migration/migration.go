// Package migration contains the elements for writing and running database
// migration operations.
package migration

// Script represents a single database migration. It is recommended to keep
// migrations short, and not to run many operation in a migration.
type Script interface {
	// Up applies the migration.
	Up(db Actions) error
	// Down undoes migration.
	Down(db Actions) error
}

// Migration is a single entry of the Migrations list. It contains the migration script
// itself, a description of the migration and (optionally) a version tag. The
// version tag should be unique, and they should be declared in order.
type Migration struct {
	Description string
	Script      Script
	VersionTag  string
}
