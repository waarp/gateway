package migration

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"
	"time"
)

//go:generate stringer -type=sqlTypeCode
type sqlTypeCode uint16

const (
	nullType sqlTypeCode = iota
	internal
	boolean
	tinyint
	smallint
	integer
	bigint
	float
	double
	varchar
	text
	date
	timestamp
	timestampz
	binary
	blob
)

type sqlType struct {
	code sqlTypeCode
	name string
	size uint64
}

// The SQL types supported by the migration engine. These values should be used
// when declaring a column or when adding a row to a table. If a database RDBMS
// does not support a specific type, it will be converted to the closest supported
// equivalent.
var (
	custom  = func(s string) sqlType { return sqlType{code: internal, name: s} }
	BOOLEAN = sqlType{code: boolean}

	TINYINT  = sqlType{code: tinyint}
	SMALLINT = sqlType{code: smallint}
	INTEGER  = sqlType{code: integer}
	BIGINT   = sqlType{code: bigint}

	FLOAT  = sqlType{code: float}
	DOUBLE = sqlType{code: double}

	VARCHAR = func(s uint64) sqlType { return sqlType{code: varchar, size: s} }
	TEXT    = sqlType{code: text}

	DATE       = sqlType{code: date}
	TIMESTAMP  = sqlType{code: timestamp}
	TIMESTAMPZ = sqlType{code: timestampz}

	BINARY = func(s uint64) sqlType { return sqlType{code: binary, size: s} }
	BLOB   = sqlType{code: blob}
)

//nolint:funlen
func (s *standardSQL) formatValueToSQL(val interface{}, sqlTyp sqlType) (string, error) {
	if valuer, ok := val.(driver.Valuer); ok {
		value, err := valuer.Value()
		if err != nil {
			return "", fmt.Errorf("failed to retrieve value from %T parameter", val)
		}
		return s.formatValueToSQL(value, sqlTyp)
	}
	typ := reflect.TypeOf(val)
	kind := typ.Kind()

	wrongType := func(exp ...string) (string, error) {
		return "", fmt.Errorf("expected value of type %s, got %T",
			strings.Join(exp, ", "), val)
	}

	wrongKind := func(kinds ...reflect.Kind) (string, error) {
		str := make([]string, len(kinds))
		for i := range kinds {
			str[i] = kinds[i].String()
		}
		return wrongType(str...)
	}

	parseTime := func(timeFormat string, makeUTC bool) (string, error) {
		ti, ok := val.(time.Time)
		if !ok {
			if kind == reflect.String { //exception for special values like 'current_time()'
				return fmt.Sprintf("%s", val), nil
			}
			return wrongType("time.Time")
		}
		if makeUTC {
			ti = ti.UTC()
		}
		return fmt.Sprintf("'%s'", ti.Format(timeFormat)), nil
	}

	convert := func(format string, expKinds ...reflect.Kind) (string, error) {
		for _, expKind := range expKinds {
			if kind == expKind {
				return fmt.Sprintf(format, val), nil
			}
		}
		return wrongKind(expKinds...)
	}

	switch sqlTyp.code {
	case boolean:
		return convert("%t", reflect.Bool)
	case tinyint:
		return convert("%d", reflect.Int8)
	case smallint:
		return convert("%d", reflect.Int16)
	case integer:
		return convert("%d", reflect.Int, reflect.Int32)
	case bigint:
		return convert("%d", reflect.Int64)
	case float:
		return convert("%f", reflect.Float32)
	case double:
		return convert("%f", reflect.Float64)
	case varchar, text:
		return convert("'%s'", reflect.String)
	case date:
		return parseTime("2006-01-02", true)
	case timestamp:
		return parseTime("2006-01-02 15:04:05.999999999", true)
	case timestampz:
		return parseTime("2006-01-02 15:04:05.999999999Z07:00", false)
	case binary:
		if typ.AssignableTo(reflect.TypeOf(0)) {
			return fmt.Sprintf("X'%X'", val), nil
		}
		fallthrough
	case blob:
		if kind != reflect.Slice && typ.Elem().Kind() != reflect.Uint8 {
			return wrongType("[]byte")
		}
		return fmt.Sprintf("X'%X'", val), nil
	default:
		return "", fmt.Errorf("unsupported SQL datatype")
	}
}

func isIntegerType(typ sqlType) bool {
	switch typ.code {
	case tinyint, smallint, integer, bigint:
		return true
	default:
		return false
	}
}
