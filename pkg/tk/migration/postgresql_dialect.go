package migration

import (
	"fmt"
	"reflect"

	// Register the PostgreSQL driver.
	_ "github.com/jackc/pgx/v4/stdlib"
)

// PostgreSQL is the constant name of the PostgreSQL dialect translator.
const PostgreSQL = "postgresql"

//nolint:gochecknoinits // init is used by design
func init() {
	dialects[PostgreSQL] = newPostgreEngine
}

type postgreTranslator struct{ standardTranslator }

func (*postgreTranslator) booleanType() string      { return "BOOL" }
func (*postgreTranslator) tinyIntType() string      { return "INT2" }
func (*postgreTranslator) smallIntType() string     { return "INT2" }
func (*postgreTranslator) integerType() string      { return "INT4" }
func (*postgreTranslator) bigIntType() string       { return "INT8" }
func (*postgreTranslator) floatType() string        { return "FLOAT4" }
func (*postgreTranslator) doubleType() string       { return "FLOAT8" }
func (*postgreTranslator) binaryType(uint64) string { return "BYTEA" }
func (*postgreTranslator) blobType() string         { return "BYTEA" }
func (*postgreTranslator) timeStampZType() string   { return "TIMESTAMPTZ" }

func (p *postgreTranslator) formatBinary(val interface{}) (string, error) {
	if typ := reflect.TypeOf(val); typ.AssignableTo(reflect.TypeOf(0)) {
		return fmt.Sprintf("'\\x%X'", val), nil
	}

	return p.formatBlob(val)
}

func (*postgreTranslator) formatBlob(val interface{}) (string, error) {
	typ := reflect.TypeOf(val)
	kind := typ.Kind()

	if kind != reflect.Slice && typ.Elem().Kind() != reflect.Uint8 {
		return wrongType(val, "[]byte")
	}

	return fmt.Sprintf("'\\x%X'", val), nil
}

func (*postgreTranslator) makeAutoIncrement(builder *tableBuilder, colType sqlType) error {
	if !isIntegerType(colType) {
		return fmt.Errorf("auto-increments can only be used on "+
			"integer types (%s is not an integer type): %w",
			colType.code.String(), errBadConstraint)
	}
	// PostgreSQL handles auto-increments by changing the type to a serial.
	//nolint:exhaustive // those are the only possible values
	switch colType.code {
	case tinyint, smallint:
		builder.getLastCol().typ = "SMALLSERIAL"
	case integer:
		builder.getLastCol().typ = "SERIAL"
	case bigint:
		builder.getLastCol().typ = "BIGSERIAL"
	}

	return nil
}

// postgreActions is the dialect engine for SQLite.
type postgreActions struct {
	*standardSQL
	trad *postgreTranslator
}

func newPostgreEngine(db *queryWriter) Actions {
	return &postgreActions{
		standardSQL: &standardSQL{queryWriter: db},
		trad:        &postgreTranslator{},
	}
}

func (*postgreActions) GetDialect() string { return PostgreSQL }

func (p *postgreActions) CreateTable(table string, defs ...Definition) error {
	return p.standardSQL.createTable(p.trad, table, defs)
}

func (p *postgreActions) AddColumn(table, column string, dataType sqlType,
	constraints ...Constraint) error {
	return p.standardSQL.addColumn(p.trad, table, column, dataType, constraints)
}

func (p *postgreActions) ChangeColumnType(table, col string, from, to sqlType) error {
	if !from.canConvertTo(to) {
		return fmt.Errorf("cannot convert from type %s to type %s: %w", from.code.String(),
			to.code.String(), errOperation)
	}

	newType, err := makeType(to, p.trad)
	if err != nil {
		return err
	}

	query := "ALTER TABLE %s\nALTER COLUMN %s TYPE %s USING %s::%s"

	return p.Exec(query, table, col, newType, col, newType)
}

func (p *postgreActions) AddRow(table string, values Cells) error {
	return p.standardSQL.addRow(p.trad, table, values)
}
