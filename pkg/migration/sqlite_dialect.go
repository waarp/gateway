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

func newSqliteEngine(db *queryWriter) Dialect {
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

func (s *sqliteDialect) makeConstraints(table string, col *Column, uniques *[][]string) (string, error) {
	var consList []string
	for _, c := range col.Constraints {
		switch c.kind {
		case primaryKey:
			consList = append(consList, fmt.Sprintf("CONSTRAINT %s_pk PRIMARY KEY", table))
		case autoIncr:
			consList = append(consList, "AUTOINCREMENT")
			col.Type = INTEGER //SQLite only accepts autoincrements on INTEGER type
		case notNull:
			consList = append(consList, "NOT NULL")
		case defaultVal:
			sqlVal, err := s.formatValueToSQL(c.params[0], col.Type)
			if err != nil {
				return "", err
			}
			consList = append(consList, fmt.Sprintf("DEFAULT %s", sqlVal))
		case unique:
			colNames, err := convertUniqueParams(col.Name, c.params)
			if err != nil {
				return "", err
			}
			if !hasEquivalent(*uniques, colNames) {
				*uniques = append(*uniques, colNames)
			}
		default:
			return "", fmt.Errorf("unknown constraint")
		}
	}
	return strings.Join(consList, " "), nil
}

func (s *sqliteDialect) makeColumnStr(table string, col Column,
	uniques *[][]string) (string, string, error) {

	constStr, err := s.makeConstraints(table, &col, uniques)
	if err != nil {
		return "", "", err
	}
	typ, err := s.sqlTypeToDBType(col.Type)
	if err != nil {
		return "", "", err
	}
	return typ, constStr, nil
}

func (s *sqliteDialect) CreateTable(table string, columns ...Column) error {
	//unique constraints are handled in a separate statement
	var uniques [][]string

	var cols []string
	for _, col := range columns {
		typ, constStr, err := s.makeColumnStr(table, col, &uniques)
		if err != nil {
			return err
		}
		if constStr == "" {
			cols = append(cols, fmt.Sprintf("%s %s", col.Name, typ))
		} else {
			cols = append(cols, fmt.Sprintf("%s %s %s", col.Name, typ, constStr))
		}
	}
	colsStr := strings.Join(cols, ", ")
	_, err := s.Exec("CREATE TABLE %s (%s)", table, colsStr)
	if err != nil {
		return err
	}

	for _, uniqueCols := range uniques {
		if _, err := s.Exec("CREATE UNIQUE INDEX %s_uindex ON %s (%s)",
			strings.Join(uniqueCols, "_"), table,
			strings.Join(uniqueCols, ", ")); err != nil {
			return err
		}
	}
	return nil
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
