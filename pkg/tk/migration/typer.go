package migration

import (
	"errors"
	"fmt"
)

var errUnknownType = errors.New("unknown SQL type")

type typer interface {
	booleanType() string
	tinyIntType() string
	smallIntType() string
	integerType() string
	bigIntType() string
	floatType() string
	doubleType() string
	varCharType(size uint64) string
	textType() string
	dateType() string
	timeStampType() string
	timeStampZType() string
	binaryType(size uint64) string
	blobType() string
}

type standardTyper struct{}

func (*standardTyper) booleanType() string    { return "BOOLEAN" }
func (*standardTyper) tinyIntType() string    { return "SMALLINT" }
func (*standardTyper) smallIntType() string   { return "SMALLINT" }
func (*standardTyper) integerType() string    { return "INTEGER" }
func (*standardTyper) bigIntType() string     { return "BIGINT" }
func (*standardTyper) floatType() string      { return "FLOAT" }
func (*standardTyper) doubleType() string     { return "DOUBLE" }
func (*standardTyper) textType() string       { return "TEXT" }
func (*standardTyper) dateType() string       { return "DATE" }
func (*standardTyper) timeStampType() string  { return "TIMESTAMP" }
func (*standardTyper) timeStampZType() string { return "TIMESTAMPZ" }
func (*standardTyper) blobType() string       { return "BLOB" }

func (*standardTyper) varCharType(size uint64) string { return fmt.Sprintf("VARCHAR(%d)", size) }
func (*standardTyper) binaryType(size uint64) string  { return fmt.Sprintf("BYTES(%d)", size) }

func makeType(typ sqlType, typer typer) (string, error) {
	switch typ.code {
	case boolean:
		return typer.booleanType(), nil
	case tinyint:
		return typer.tinyIntType(), nil
	case smallint:
		return typer.smallIntType(), nil
	case integer:
		return typer.integerType(), nil
	case bigint:
		return typer.bigIntType(), nil
	case float:
		return typer.floatType(), nil
	case double:
		return typer.doubleType(), nil
	case varchar:
		return typer.varCharType(typ.size), nil
	case text:
		return typer.textType(), nil
	case date:
		return typer.dateType(), nil
	case timestamp:
		return typer.timeStampType(), nil
	case timestampz:
		return typer.timeStampZType(), nil
	case binary:
		return typer.binaryType(typ.size), nil
	case blob:
		return typer.blobType(), nil
	default:
		return "", fmt.Errorf("%w '%s'", errUnknownType, typ.code.String())
	}
}
