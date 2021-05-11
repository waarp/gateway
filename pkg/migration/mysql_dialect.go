package migration

import (
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql" //register the MySQL driver
)

// MySQL is the constant name of the MySQL dialect translator.
const MySQL = "mysql"

func init() {
	dialects[MySQL] = newMySQLEngine
}

// mySQLDialect is the dialect engine for SQLite.
type mySQLDialect struct{ *standardSQL }

func newMySQLEngine(db *queryWriter) Dialect {
	return &mySQLDialect{standardSQL: &standardSQL{queryWriter: db}}
}

func (m *mySQLDialect) sqlTypeToDBType(typ sqlType) (string, error) {
	switch typ.code {
	case boolean, tinyint:
		return "TINYINT", nil
	case smallint:
		return "SMALLINT", nil
	case integer:
		return "INT", nil
	case bigint:
		return "BIGINT", nil
	case float:
		return "FLOAT", nil
	case double:
		return "DOUBLE", nil
	case varchar:
		return fmt.Sprintf("VARCHAR(%d)", typ.size), nil
	case text:
		return "TEXT", nil
	case date:
		return "DATE", nil
	case timestamp:
		return "TIMESTAMP", nil
	case timestampz:
		return "TEXT", nil
	case binary:
		return fmt.Sprintf("BINARY(%d)", typ.size), nil
	case blob:
		return "BLOB", nil
	default:
		return "", fmt.Errorf("unsupported SQL datatype")
	}
}

func (m *mySQLDialect) makeConstraints(table string, col *Column,
	uniques *[][]string) (string, string, error) {

	var consList []string
	var pk string
	for _, c := range col.Constraints {
		switch c.kind {
		case primaryKey:
			pk = fmt.Sprintf("CONSTRAINT %s_pk PRIMARY KEY (%s)", table, col.Name)
		case autoIncr:
			consList = append(consList, "AUTO_INCREMENT")
		case notNull:
			consList = append(consList, "NOT NULL")
		case defaultVal:
			sqlVal, err := m.formatValueToSQL(c.params[0], col.Type)
			if err != nil {
				return "", "", err
			}
			consList = append(consList, fmt.Sprintf("DEFAULT %s", sqlVal))
		case unique:
			colNames, err := convertUniqueParams(col.Name, c.params)
			if err != nil {
				return "", "", err
			}
			if !hasEquivalent(*uniques, colNames) {
				*uniques = append(*uniques, colNames)
			}
		default:
			return "", "", fmt.Errorf("unknown constraint")
		}
	}
	return strings.Join(consList, " "), pk, nil
}

func (m *mySQLDialect) makeColumnStr(table string, col Column,
	uniques *[][]string) (string, string, string, error) {

	constStr, pkStr, err := m.makeConstraints(table, &col, uniques)
	if err != nil {
		return "", "", "", err
	}
	typ, err := m.sqlTypeToDBType(col.Type)
	if err != nil {
		return "", "", "", err
	}
	return typ, constStr, pkStr, nil
}

func (m *mySQLDialect) CreateTable(table string, columns ...Column) error {
	if len(columns) == 0 {
		return fmt.Errorf("missing columns in CREATE TABLE statement")
	}

	//unique constraints are handled in a separate statement
	var uniques [][]string

	var cols []string
	for _, col := range columns {
		typ, constStr, pkStr, err := m.makeColumnStr(table, col, &uniques)
		if err != nil {
			return err
		}
		if constStr == "" {
			cols = append(cols, fmt.Sprintf("%s %s", col.Name, typ))
		} else {
			cols = append(cols, fmt.Sprintf("%s %s %s", col.Name, typ, constStr))
		}
		if pkStr != "" {
			cols = append(cols, pkStr)
		}
	}
	colsStr := strings.Join(cols, ", ")
	_, err := m.Exec("CREATE TABLE %s (%s)", table, colsStr)
	if err != nil {
		return err
	}

	for _, uniqueCols := range uniques {
		if _, err := m.Exec("CREATE UNIQUE INDEX %s_uindex ON %s (%s)",
			strings.Join(uniqueCols, "_"), table,
			strings.Join(uniqueCols, ", ")); err != nil {
			return err
		}
	}
	return nil
}
