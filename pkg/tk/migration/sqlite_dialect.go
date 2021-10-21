package migration

import (
	"fmt"

	// Register the SQLite driver.
	_ "github.com/mattn/go-sqlite3"
)

// SQLite is the constant name of the SQLite dialect translator.
const SQLite = "sqlite"

//nolint:gochecknoinits // init is used by design
func init() {
	dialects[SQLite] = newSqliteEngine
}

type sqliteError string

func (s sqliteError) Error() string { return string(s) }

// sqliteDialect is the dialect engine for SQLite.
type sqliteDialect struct{ *standardSQL }

func newSqliteEngine(db *queryWriter) Actions {
	return &sqliteDialect{standardSQL: &standardSQL{queryWriter: db}}
}

func (*sqliteDialect) GetDialect() string { return SQLite }

/*
func (s *sqliteDialect) isAtLeastVersion(target string) (bool, error) {
	res := s.QueryRow("SELECT sqlite_version()")

	var version string
	if err := res.Scan(&version); err != nil {
		return false, fmt.Errorf("failed to retrieve SQLite version: %w", err)
	}

	version = "v" + version
	if !semver.IsValid(version) {
		return false, fmt.Errorf("failed to parse SQLite version: '%s' is not a valid version",
			version)
	}

	return semver.Compare(version, target) >= 0, nil
}
*/

func (s *sqliteDialect) sqlTypeToDBType(typ sqlType) (string, error) {
	switch typ.code {
	case boolean, tinyint, smallint, integer, bigint:
		return "INTEGER", nil
	case float, double:
		return "REAL", nil
	case varchar, text, timestampz:
		return "TEXT", nil
	case date, timestamp:
		return "NUMERIC", nil
	case binary, blob:
		return "BLOB", nil
	default:
		return "", fmt.Errorf("unsupported SQL datatype") //nolint:goerr113 // base error
	}
}

//nolint:dupl // FIXME to refactor
func (s *sqliteDialect) makeConstraints(col *Column) ([]string, error) {
	var consList []string

	for _, c := range col.Constraints {
		switch con := c.(type) {
		case pk:
			consList = append(consList, "PRIMARY KEY")
		case fk:
			consList = append(consList, fmt.Sprintf("REFERENCES %s(%s)", con.table, con.col))
		case notNull:
			consList = append(consList, "NOT NULL")
		case autoIncr:
			if !isIntegerType(col.Type) {
				return nil, fmt.Errorf("auto-increments can only be used on "+
					"integer types (%s is not an integer type): %w",
					col.Type.code.String(), errBadConstraint)
			}

			consList = append(consList, "AUTOINCREMENT")

		case unique:
			consList = append(consList, "UNIQUE")
		case defaul:
			sqlVal, err := s.formatValueToSQL(con.val, col.Type)
			if err != nil {
				return nil, err
			}

			consList = append(consList, fmt.Sprintf("DEFAULT %s", sqlVal))

		default:
			return nil, fmt.Errorf("unknown constraint type %T: %w", c, errBadConstraint)
		}
	}

	return consList, nil
}

func (s *sqliteDialect) CreateTable(table string, defs ...Definition) error {
	return s.standardSQL.createTable(s, table, defs)
}

func (s *sqliteDialect) ChangeColumnType(_, _ string, from, to sqlType) error {
	if from.canConvertTo(to) {
		return nil // nothing to do
	}

	return fmt.Errorf("cannot convert from type %s to type %s: %w",
		from.code.String(), to.code.String(), errOperation)
}

func (s *sqliteDialect) AddRow(table string, values Cells) error {
	return s.addRow(s, table, values)
}

func (s *sqliteDialect) AddColumn(table, column string, dataType sqlType,
	constraints ...Constraint) error {
	return s.standardSQL.addColumn(s, table, column, dataType, constraints)
}
