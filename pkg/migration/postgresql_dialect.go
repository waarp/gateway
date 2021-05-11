package migration

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"

	_ "github.com/jackc/pgx/v4/stdlib" //register the PostgreSQL driver
)

// PostgreSQL is the constant name of the PostgreSQL dialect translator.
const PostgreSQL = "postgresql"

func init() {
	dialects[PostgreSQL] = newPostgreEngine
}

// postgreDialect is the dialect engine for SQLite.
type postgreDialect struct{ *standardSQL }

func newPostgreEngine(db *queryWriter) Dialect {
	return &postgreDialect{standardSQL: &standardSQL{queryWriter: db}}
}

func (p *postgreDialect) formatValueToSQL(val interface{}, sqlTyp sqlType) (string, error) {
	if valuer, ok := val.(driver.Valuer); ok {
		value, err := valuer.Value()
		if err != nil {
			return "", err
		}
		return p.formatValueToSQL(value, sqlTyp)
	}

	typ := reflect.TypeOf(val)
	kind := typ.Kind()
	switch sqlTyp.code {
	case internal:
		return fmt.Sprint(val), nil
	case binary:
		if typ.AssignableTo(reflect.TypeOf(0)) {
			return fmt.Sprintf("'\\x%X'", val), nil
		}
		fallthrough
	case blob:
		if kind != reflect.Slice && typ.Elem().Kind() != reflect.Uint8 {
			return "", fmt.Errorf("expected value of type []byte, got %T", val)
		}
		return fmt.Sprintf("'\\x%X'", val), nil
	default:
		return p.standardSQL.formatValueToSQL(val, sqlTyp)
	}
}

func (p *postgreDialect) sqlTypeToDBType(typ sqlType) (string, error) {
	switch typ.code {
	case internal:
		return typ.name, nil
	case boolean:
		return "BOOL", nil
	case tinyint, smallint:
		return "INT2", nil
	case integer:
		return "INT4", nil
	case bigint:
		return "INT8", nil
	case float:
		return "FLOAT4", nil
	case double:
		return "FLOAT8", nil
	case varchar:
		return fmt.Sprintf("VARCHAR(%d)", typ.size), nil
	case text:
		return "TEXT", nil
	case date:
		return "DATE", nil
	case timestamp:
		return "TIMESTAMP", nil
	case timestampz:
		return "TIMESTAMPTZ", nil
	case binary, blob:
		return "BYTEA", nil
	default:
		return "", fmt.Errorf("unsupported SQL datatype")
	}
}

func (p *postgreDialect) makeConstraints(table string, col *Column, uniques *[][]string) (string, error) {
	var consList []string
	for _, c := range col.Constraints {
		switch c.kind {
		case primaryKey:
			consList = append(consList, fmt.Sprintf("CONSTRAINT %s_pk PRIMARY KEY", table))
		case autoIncr:
			switch col.Type.code {
			case tinyint, smallint:
				col.Type = custom("SMALLSERIAL")
			case integer:
				col.Type = custom("SERIAL")
			case bigint:
				col.Type = custom("BIGSERIAL")
			}
		case notNull:
			consList = append(consList, "NOT NULL")
		case defaultVal:
			sqlVal, err := p.formatValueToSQL(c.params[0], col.Type)
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

func (p *postgreDialect) makeColumnStr(table string, col Column,
	uniques *[][]string) (string, string, error) {

	constStr, err := p.makeConstraints(table, &col, uniques)
	if err != nil {
		return "", "", err
	}
	typ, err := p.sqlTypeToDBType(col.Type)
	if err != nil {
		return "", "", err
	}
	return typ, constStr, nil
}

func (p *postgreDialect) CreateTable(table string, columns ...Column) error {
	if len(columns) == 0 {
		return fmt.Errorf("missing columns in CREATE TABLE statement")
	}

	//unique constraints are handled in a separate statement
	var uniques [][]string

	var cols []string
	for _, col := range columns {
		typ, constStr, err := p.makeColumnStr(table, col, &uniques)
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
	_, err := p.Exec("CREATE TABLE %s (%s)", table, colsStr)
	if err != nil {
		return err
	}

	for _, uniqueCols := range uniques {
		if _, err := p.Exec("CREATE UNIQUE INDEX %s_uindex ON %s (%s)",
			strings.Join(uniqueCols, "_"), table,
			strings.Join(uniqueCols, ", ")); err != nil {
			return err
		}
	}
	return nil
}

func (p *postgreDialect) AddRow(table string, values Cells) error {
	return p.standardSQL.addRow(p.formatValueToSQL, table, values)
}
