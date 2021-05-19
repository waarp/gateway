package migration

import (
	"database/sql"
	"fmt"
	"io"
)

// QueryExecutor is an interface exposing the 2 common functions for executing
// SQL requests in the standard SQL library. However, unlike the standard library,
// QueryExecutor uses the fmt placeholder verbs instead of the database's verbs.
type QueryExecutor interface {
	Exec(string, ...interface{}) (sql.Result, error)
	Query(string, ...interface{}) (*sql.Rows, error)
}

// Wrapper for QueryExecutor which optionally takes a strings.Builder. If a Builder
// is given, calls to Exec will write the query to the Builder instead of sending
// it to the database. Used for extracting the migration script.
type queryWriter struct {
	db     QueryExecutor
	writer io.Writer
}

func (q *queryWriter) Exec(query string, args ...interface{}) (sql.Result, error) {
	command := fmt.Sprintf(query, args...)
	if q.writer != nil {
		_, err := q.writer.Write([]byte(fmt.Sprintf("\n%s;\n", command)))
		if err != nil {
			return nil, err
		}
		if !isTest {
			return nil, nil
		}
	}

	return q.db.Exec(command)
}

func (q *queryWriter) Query(query string, args ...interface{}) (*sql.Rows, error) {
	command := fmt.Sprintf(query, args...)
	return q.db.Query(command)
}

var dialects = map[string]func(*queryWriter) Actions{}

// Definition represents one part of a column definition. It can be a column
// or a table constraint.
type Definition interface{}

// Actions is an interface exposing the standard Exec & Query functions for
// sending queries to a database, along with a few helper functions for common
// operations. Actions is used for database migration operations.
type Actions interface {
	QueryExecutor
	DestructiveActions

	// CreateTable creates a new table with the given definitions. A definition
	// can be either Column or a TableConstraint.
	CreateTable(name string, defs ...Definition) error

	// AddColumn adds a new column
	AddColumn(table, column string, dataType sqlType, constraints ...Constraint) error

	// AddRow inserts a new row into the given table.
	AddRow(table string, values Cells) error
}

// DestructiveActions regroups all the destructive migrations actions. Since they
// are destructive, these actions should only be used during major updates, as
// they break retro-compatibility.
type DestructiveActions interface {
	// RenameTable changes the name of the given table to a new one.
	RenameTable(oldName, newName string) error

	// DropTable drops the given table.
	DropTable(name string) error

	// RenameColumn changes the name of the given column.
	RenameColumn(table, oldName, newName string) error

	// ChangeColumnType changes the type of the given column.
	ChangeColumnType(table, col string, old, new sqlType) error

	// DropColumn drops the given column.
	DropColumn(table, name string) error
}
