package migration

import (
	"errors"
	"fmt"
	"strings"

	// Register the SQLite driver.
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/mod/semver"
)

var errSqliteVersion = errors.New("cannot get sqlite version")

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

func (s *sqliteDialect) isAtLeastVersion(target string) (bool, error) {
	res, err := s.Query("SELECT sqlite_version()")
	if err != nil {
		return false, fmt.Errorf("failed to run a query to retrieve SQLite version: %w", err)
	}

	defer res.Close() //nolint:errcheck // no logger to handle the error

	var version string
	if !res.Next() || res.Scan(&version) != nil {
		return false, fmt.Errorf("failed to retrieve SQLite version: %w", errSqliteVersion)
	}

	version = "v" + version
	if !semver.IsValid(version) {
		return false, fmt.Errorf("failed to parse SQLite version: '%s' is not a valid version: %w",
			version, err)
	}

	return semver.Compare(version, target) >= 0, nil
}

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

func (s *sqliteDialect) DropColumn(table, name string) error {
	ok, err := s.isAtLeastVersion("v3.35.0")
	if err != nil {
		return err
	}

	// DROP COLUMN is supported on SQLite versions >= 3.35
	if !isTest && ok {
		return s.standardSQL.DropColumn(table, name)
	}
	// otherwise, we have to create a new table without the column, and copy the content

	cols, err := getColumnsNames(s, table)
	if err != nil {
		return err
	}

	var found bool

	for i, col := range cols {
		if col == name {
			cols = append(cols[:i], cols[i+1:]...)
			found = true

			break
		}
	}

	if !found {
		return sqliteError(fmt.Sprintf(`no such column: "%s"`, name))
	}

	query := fmt.Sprintf("CREATE TABLE " + table + "_new AS SELECT " +
		strings.Join(cols, ", ") + " FROM " + table)

	if err := s.Exec(query); err != nil {
		return err
	}

	if err := s.DropTable(table); err != nil {
		return err
	}

	if err := s.RenameTable(table+"_new", table); err != nil {
		return err
	}

	return nil
}
