package migration

import (
	"database/sql/driver"
	"fmt"
	"reflect"

	_ "github.com/jackc/pgx/v4/stdlib" //register the PostgreSQL driver
)

// PostgreSQL is the constant name of the PostgreSQL dialect translator.
const PostgreSQL = "postgresql"

func init() {
	dialects[PostgreSQL] = newPostgreEngine
}

// postgreDialect is the dialect engine for SQLite.
type postgreDialect struct{ *standardSQL }

func newPostgreEngine(db *queryWriter) Actions {
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

func (p *postgreDialect) makeConstraints(col *Column) ([]string, error) {
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
			// PostgreSQL handles auto-increments by changing the type to a serial.
			switch col.Type.code {
			case tinyint, smallint:
				col.Type = custom("SMALLSERIAL")
			case integer:
				col.Type = custom("SERIAL")
			case bigint:
				col.Type = custom("BIGSERIAL")
			}
		case unique:
			consList = append(consList, "UNIQUE")
		case defaul:
			sqlVal, err := p.formatValueToSQL(con.val, col.Type)
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

func (p *postgreDialect) CreateTable(table string, defs ...Definition) error {
	return p.standardSQL.createTable(p, table, defs)
}

func (p *postgreDialect) AddColumn(table, column string, dataType sqlType,
	constraints ...Constraint) error {
	return p.standardSQL.addColumn(p, table, column, dataType, constraints)
}

func (p *postgreDialect) ChangeColumnType(table, col string, old, new sqlType) error {
	if !old.canConvertTo(new) {
		return fmt.Errorf("cannot convert from type %s to type %s", old.code.String(),
			new.code.String())
	}

	newType, err := p.sqlTypeToDBType(new)
	if err != nil {
		return err
	}

	query := "ALTER TABLE %s\nALTER COLUMN %s TYPE %s USING %s::%s"
	_, err = p.Exec(query, table, col, newType, col, newType)
	return err
}

func (p *postgreDialect) AddRow(table string, values Cells) error {
	return p.standardSQL.addRow(p, table, values)
}
