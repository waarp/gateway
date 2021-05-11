package migration

import "fmt"

// isTest indicates whether the current environment is a test environnement or not.
// Should be used to run specific tests, even when the normal conditions are not met
// (like the test database's version).
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
	Constraints []constraint
}

// Col is a shortcut function for instantiating a Column without having to declare
// the attributes' names.
func Col(name string, typ sqlType, constraints ...constraint) Column {
	return Column{Name: name, Type: typ, Constraints: constraints}
}

func getColumnsNames(db QueryExecutor, table string) ([]string, error) {
	rows, err := db.Query("SELECT * FROM %s", table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return rows.Columns()
}

func hasEquivalent(list [][]string, elem []string) bool {
	isEquivalent := func(l1, l2 []string) bool {
		if len(l1) != len(l2) {
			return false
		}
		contains := func(l []string, e string) bool {
			for i := range l {
				if l[i] == e {
					return true
				}
			}
			return false
		}

		for _, e := range l2 {
			if !contains(l1, e) {
				return false
			}
		}
		return true
	}

	for _, slice := range list {
		if isEquivalent(slice, elem) {
			return true
		}
	}
	return false
}

func convertUniqueParams(col string, elem []interface{}) ([]string, error) {
	str := make([]string, len(elem))
	for i := range elem {
		s, ok := elem[i].(string)
		if !ok {
			return nil, fmt.Errorf("invalid UNIQUE parameter, expected a string, got a %T", elem[i])
		}
		str[i] = s
	}
	return append(str, col), nil
}
