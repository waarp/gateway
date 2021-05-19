package migration

import (
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3" //register the SQLite driver
	"golang.org/x/mod/semver"
)

// SQLite is the constant name of the SQLite dialect translator.
const SQLite = "sqlite"

func init() {
	dialects[SQLite] = newSqliteEngine
}

// sqliteDialect is the dialect engine for SQLite.
type sqliteDialect struct{ *standardSQL }

func newSqliteEngine(db *queryWriter) Actions {
	return &sqliteDialect{standardSQL: &standardSQL{queryWriter: db}}
}

func (s *sqliteDialect) isAtLeastVersion(target string) (bool, error) {
	res, err := s.Query("SELECT sqlite_version()")
	if err != nil {
		return false, fmt.Errorf("failed to retrieve SQLite version")
	}
	defer res.Close()

	var version string
	if !res.Next() || res.Scan(&version) != nil {
		return false, fmt.Errorf("failed to retrieve SQLite version")
	}

	version = "v" + version
	if !semver.IsValid(version) {
		return false, fmt.Errorf("failed to parse SQLite version: '%s' is not a valid version", version)
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
		return "", fmt.Errorf("unsupported SQL datatype")
	}
}

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
					"integer types (%s is not an integer type)", col.Type.code.String())
			}
			// AUTOINCR is not needed in SQLite, INTEGER PRIMARY KEY column already
			// have an auto-increment by default.
			//consList = append(consList, "AUTOINCREMENT")
		case unique:
			consList = append(consList, "UNIQUE")
		case defaul:
			sqlVal, err := s.formatValueToSQL(con.val, col.Type)
			if err != nil {
				return nil, err
			}
			consList = append(consList, fmt.Sprintf("DEFAULT %s", sqlVal))
		default:
			return nil, fmt.Errorf("unknown constraint type %T", c)
		}
	}
	return consList, nil
}

func (s *sqliteDialect) CreateTable(table string, defs ...Definition) error {
	return s.standardSQL.createTable(s, table, defs)
}

func (s *sqliteDialect) ChangeColumnType(_, _ string, old, new sqlType) error {
	if old.canConvertTo(new) {
		return nil //nothing to do
	}
	return fmt.Errorf("cannot convert from type %s to type %s", old.code.String(),
		new.code.String())
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
	for i, col := range cols {
		if col == name {
			cols = append(cols[:i], cols[i+1:]...)
			break
		}
	}

	query := fmt.Sprintf("CREATE TABLE %s_new AS SELECT %s FROM %s", table,
		strings.Join(cols, ", "), table)

	if _, err := s.Exec(query); err != nil {
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
