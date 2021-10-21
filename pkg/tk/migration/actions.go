package migration

import (
	"database/sql"
	"errors"
)

var errOperation = errors.New("an operation failed")

//nolint:gochecknoglobals // global var is used by design
var dialects = map[string]func(*queryWriter) Actions{}

// Definition represents one part of a column definition. It can be a column
// or a table constraint.
type Definition interface{}

// Constraint represents a single SQL column constraint to be used when declaring
// a column. Valid constraints are: PrimaryKey, ForeignKey, NotNull, AutoIncr,
// Unique and Default.
type Constraint interface{}

// TableConstraint represents a constraint put on an SQL table when declaring said
// table. Valid table constraints are: MultiPrimaryKey and MultiUnique.
type TableConstraint interface{}

// Querier is an interface exposing the common function for sending database
// queries in the standard SQL library.
type Querier interface {
	Query(string, ...interface{}) (*sql.Rows, error)
	QueryRow(string, ...interface{}) *sql.Row
}

// Executor is an interface exposing the common function for executing database
// commands in the standard SQL library. However, unlike the standard library,
// Executor uses the fmt placeholder verbs instead of the database's verbs.
type Executor interface {
	Exec(string, ...interface{}) error
}

// Actions is an interface regrouping all the.
type Actions interface {
	Querier
	Executor

	// GetDialect returns the database's dialect.
	GetDialect() string

	// CreateTable creates a new table with the given definitions. A definition
	// can be either Column or a TableConstraint.
	CreateTable(name string, defs ...Definition) error

	// AddColumn adds a new column to the table with the given type and constraints.
	// It is up to the migration script to then fill that column if necessary.
	AddColumn(table, column string, dataType sqlType, constraints ...Constraint) error

	// AddRow inserts a new row into the given table.
	AddRow(table string, values Cells) error

	// RenameTable changes the name of the given table to a new one. This is a
	// destructive (non retro-compatible) operation.
	RenameTable(oldName, newName string) error

	// DropTable drops the given table. This is a destructive (non
	// retro-compatible) operation.
	DropTable(name string) error

	// RenameColumn changes the name of the given column. This is a destructive
	// (non retro-compatible) operation.
	RenameColumn(table, oldName, newName string) error

	// ChangeColumnType changes the type of the given column. This is a
	// destructive (non retro-compatible) operation.
	ChangeColumnType(table, col string, from, to sqlType) error

	// SwapColumns swaps all the values of col1 and col2. The columns' types MUST
	// be compatible for this operation to work.
	SwapColumns(table, col1, col2, cond string) error

	// DropColumn drops the given column. This is a destructive (non
	// retro-compatible) operation.
	DropColumn(table, name string) error
}
