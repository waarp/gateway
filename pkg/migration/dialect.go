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
		_, err := q.writer.Write(append([]byte(command), ";\n"...))
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

var dialects = map[string]func(*queryWriter) Dialect{}

// Dialect is an interface exposing the standard Exec & Query functions for
// sending queries to a database, along with a few helper functions for common
// operations. Dialect is used for database migration operations.
type Dialect interface {
	QueryExecutor

	sqlTypeToDBType(sqlType sqlType) (string, error)

	CreateTable(name string, columns ...Column) error
	RenameTable(oldName, newName string) error
	DropTable(name string) error

	RenameColumn(table, oldName, newName string) error
	AddColumn(table, name, datatype string) error
	DropColumn(table, name string) error

	AddRow(table string, values Cells) error
}
