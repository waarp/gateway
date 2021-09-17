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

func (s1 sqlType) canConvertTo(s2 sqlType) bool {
	if s1.code == s2.code {
		return true
	}

	//nolint:exhaustive // missing cases are handled by the final return
	switch s1.code {
	case tinyint:
		switch s2.code {
		case smallint, integer, bigint:
			return true
		}
	case smallint:
		switch s2.code {
		case integer, bigint:
			return true
		}
	case integer:
		if s2.code == bigint {
			return true
		}
	case float:
		if s2.code == double {
			return true
		}
	case varchar:
		if s2.code == text {
			return true
		}
	}

	return false
}

// The SQL types supported by the migration engine. These values should be used
// when declaring a column or when adding a row to a table. If a database RDBMS
// does not support a specific type, it will be converted to the closest supported
// equivalent.
//nolint:gochecknoglobals // global var is used by design
var (
	custom  = func(s string) sqlType { return sqlType{code: internal, name: s} }
	Boolean = sqlType{code: boolean}

	TinyInt  = sqlType{code: tinyint}
	SmallInt = sqlType{code: smallint}
	Integer  = sqlType{code: integer}
	BigInt   = sqlType{code: bigint}

	Float  = sqlType{code: float}
	Double = sqlType{code: double}

	Varchar = func(s uint64) sqlType { return sqlType{code: varchar, size: s} }
	Text    = sqlType{code: text}

	Date       = sqlType{code: date}
	Timestamp  = sqlType{code: timestamp}
	Timestampz = sqlType{code: timestampz}

	Binary = func(s uint64) sqlType { return sqlType{code: binary, size: s} }
	Blob   = sqlType{code: blob}
)

func (s *standardSQL) formatValueToSQL(val interface{}, sqlTyp sqlType) (string, error) {
	if valuer, ok := val.(driver.Valuer); ok {
		value, err := valuer.Value()
		if err != nil {
			return "", fmt.Errorf("failed to retrieve value from %T parameter: %w", val, err)
		}

		return s.formatValueToSQL(value, sqlTyp)
	}

	typ := reflect.TypeOf(val)
	kind := typ.Kind()

	switch sqlTyp.code {
	case boolean:
		return convert(val, kind, "%t", reflect.Bool)
	case tinyint:
		return convert(val, kind, "%d", reflect.Int8)
	case smallint:
		return convert(val, kind, "%d", reflect.Int16)
	case integer:
		return convert(val, kind, "%d", reflect.Int, reflect.Int32)
	case bigint:
		return convert(val, kind, "%d", reflect.Int64)
	case float:
		return convert(val, kind, "%f", reflect.Float32)
	case double:
		return convert(val, kind, "%f", reflect.Float64)
	case varchar, text:
		return convert(val, kind, "'%s'", reflect.String)
	case date:
		return parseTime(val, kind, "2006-01-02", true)
	case timestamp:
		return parseTime(val, kind, "2006-01-02 15:04:05.999999999", true)
	case timestampz:
		return parseTime(val, kind, "2006-01-02 15:04:05.999999999Z07:00", false)
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
		return "", fmt.Errorf("unsupported SQL datatype") //nolint:goerr113 // no need here
	}
}

func wrongType(val interface{}, exp ...string) (string, error) {
	//nolint:goerr113 // no need here
	return "", fmt.Errorf("expected value of type %s, got %T",
		strings.Join(exp, ", "), val)
}

func wrongKind(val interface{}, kinds ...reflect.Kind) (string, error) {
	str := make([]string, len(kinds))
	for i := range kinds {
		str[i] = kinds[i].String()
	}

	return wrongType(val, str...)
}

func parseTime(val interface{}, kind reflect.Kind, timeFormat string, makeUTC bool) (string, error) {
	ti, ok := val.(time.Time)
	if !ok {
		if kind == reflect.String { // exception for special values like 'current_time()'
			return fmt.Sprintf("%s", val), nil
		}

		return wrongType("time.Time")
	}

	if makeUTC {
		ti = ti.UTC()
	}

	return fmt.Sprintf("'%s'", ti.Format(timeFormat)), nil
}

func convert(val interface{}, kind reflect.Kind, format string, expKinds ...reflect.Kind) (string, error) {
	for _, expKind := range expKinds {
		if kind == expKind {
			return fmt.Sprintf(format, val), nil
		}
	}

	return wrongKind(val, expKinds...)
}

func isIntegerType(typ sqlType) bool {
	switch typ.code {
	case tinyint, smallint, integer, bigint:
		return true
	default:
		return false
	}
}
