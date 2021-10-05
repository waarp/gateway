package migration

import "fmt"

// isTest indicates whether the current environment is a test environnement or not.
// Should be used to run specific tests, even when the normal conditions are not met
// (like the test database's version).
//nolint:gochecknoglobals // global var is used by design
var isTest bool

// Cells is a map type representing a table row in a INSERT INTO statement. It
// associates each column's name with its value for that row.
type Cells map[string]Cell

// Cell is the type representing a single value in the row. The value's type is
// also required for type checking purposes.
type Cell struct {
	Val  interface{}
	Type sqlType
}

// Cel is a shortcut function for instantiating a Cell without having to declare
// the attributes' names.
func Cel(typ sqlType, val interface{}) Cell {
	return Cell{Val: val, Type: typ}
}

// Column is a type representing a column declaration in a CREATE TABLE statement.
// It contains the column's name, type and its constraints (if it has some).
type Column struct {
	Name        string
	Type        sqlType
	Constraints []Constraint
}

// Col is a shortcut function for instantiating a column without having to declare
// the attributes' names.
func Col(name string, typ sqlType, constraints ...Constraint) Column {
	return Column{Name: name, Type: typ, Constraints: constraints}
}

func getColumnsNames(db Querier, table string) ([]string, error) {
	rows, err := db.Query("SELECT * FROM %s", table)
	if err != nil {
		return nil, fmt.Errorf("cannot get column names: %w", err)
	}

	defer rows.Close() //nolint:errcheck // no logger to handle the error

	names, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("cannot get column names: %w", err)
	}

	return names, nil
}
