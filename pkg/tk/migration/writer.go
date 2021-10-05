package migration

import (
	"database/sql"
	"fmt"
	"io"
)

// Can be an *sql.DB or *sql.Tx.
type dbInterface interface {
	Query(string, ...interface{}) (*sql.Rows, error)
	Exec(string, ...interface{}) (sql.Result, error)
}

// Wrapper for Querier & Executor which optionally takes a strings.Builder. If
// a Builder is given, calls to Exec will write the query to the Builder instead
// of sending it to the database. Used for extracting the migration script.
type queryWriter struct {
	db     dbInterface
	writer io.Writer
}

func (q *queryWriter) Exec(query string, args ...interface{}) error {
	command := fmt.Sprintf(query, args...)
	if q.writer != nil {
		_, err := q.writer.Write([]byte(fmt.Sprintf("\n%s;\n", command)))
		if err != nil {
			return fmt.Errorf("cannot write command: %w", err)
		}

		if !isTest {
			return nil
		}
	}

	if _, err := q.db.Exec(command); err != nil {
		return fmt.Errorf("cannot execute command: %w", err)
	}

	return nil
}

func (q *queryWriter) Query(query string, args ...interface{}) (*sql.Rows, error) {
	command := fmt.Sprintf(query, args...)

	rows, err := q.db.Query(command)
	if err != nil {
		return nil, fmt.Errorf("cannot execute query: %w", err)
	}

	return rows, nil
}
