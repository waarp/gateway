package migration

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

//go:generate stringer -type=sqlTypeCode
type sqlTypeCode uint16

const (
	nullType sqlTypeCode = iota
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

func wrongType(val interface{}, exp ...string) (string, error) {
	//nolint:goerr113 // no need here
	return "", fmt.Errorf("failed to format value '%v': expected value of type %s",
		val, strings.Join(exp, ", "))
}

func wrongKind(val interface{}, kinds ...reflect.Kind) (string, error) {
	str := make([]string, len(kinds))
	for i := range kinds {
		str[i] = kinds[i].String()
	}

	return wrongType(val, str...)
}

func parseTime(val interface{}, timeFormat string, makeUTC bool) (string, error) {
	kind := reflect.TypeOf(val).Kind()

	ti, ok := val.(time.Time)
	if !ok {
		if kind == reflect.String { // exception for special values like 'current_time()'
			return fmt.Sprintf("%s", val), nil
		}

		return wrongType(val, "time.Time")
	}

	if makeUTC {
		ti = ti.UTC()
	}

	return fmt.Sprintf("'%s'", ti.Format(timeFormat)), nil
}

func convert(val interface{}, format string, expKinds ...reflect.Kind) (string, error) {
	kind := reflect.TypeOf(val).Kind()

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
