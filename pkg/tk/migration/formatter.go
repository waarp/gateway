package migration

import (
	"database/sql/driver"
	"fmt"
	"reflect"
)

type valueFormatter interface {
	formatBoolean(val interface{}) (string, error)
	formatTinyInt(val interface{}) (string, error)
	formatSmallInt(val interface{}) (string, error)
	formatInteger(val interface{}) (string, error)
	formatBigInt(val interface{}) (string, error)
	formatFloat(val interface{}) (string, error)
	formatDouble(val interface{}) (string, error)
	formatVarChar(val interface{}) (string, error)
	formatText(val interface{}) (string, error)
	formatDate(val interface{}) (string, error)
	formatTimeStamp(val interface{}) (string, error)
	formatTimeStampZ(val interface{}) (string, error)
	formatBinary(val interface{}) (string, error)
	formatBlob(val interface{}) (string, error)
}

//nolint:wrapcheck // wrapping errors would hurt readability without adding any value to the errors themselves
func formatValue(val interface{}, colType sqlType, formatter valueFormatter) (string, error) {
	if valuer, ok := val.(driver.Valuer); ok {
		value, err := valuer.Value()
		if err != nil {
			return "", fmt.Errorf("failed to retrieve value from %T parameter: %w", val, err)
		}

		return formatValue(value, colType, formatter)
	}

	switch colType.code {
	case boolean:
		return formatter.formatBoolean(val)
	case tinyint:
		return formatter.formatTinyInt(val)
	case smallint:
		return formatter.formatSmallInt(val)
	case integer:
		return formatter.formatInteger(val)
	case bigint:
		return formatter.formatBigInt(val)
	case float:
		return formatter.formatFloat(val)
	case double:
		return formatter.formatDouble(val)
	case varchar:
		return formatter.formatVarChar(val)
	case text:
		return formatter.formatText(val)
	case date:
		return formatter.formatDate(val)
	case timestamp:
		return formatter.formatTimeStamp(val)
	case timestampz:
		return formatter.formatTimeStampZ(val)
	case binary:
		return formatter.formatBinary(val)
	case blob:
		return formatter.formatBlob(val)
	default:
		return "", fmt.Errorf("cannot format value '%v': %w", val, errUnknownType)
	}
}

type standardFormatter struct{}

func (*standardFormatter) formatBoolean(val interface{}) (string, error) {
	return convert(val, "%t", reflect.Bool)
}

func (*standardFormatter) formatTinyInt(val interface{}) (string, error) {
	return convert(val, "%d", reflect.Int8)
}

func (*standardFormatter) formatSmallInt(val interface{}) (string, error) {
	return convert(val, "%d", reflect.Int16, reflect.Int8)
}

func (*standardFormatter) formatInteger(val interface{}) (string, error) {
	return convert(val, "%d", reflect.Int, reflect.Int32, reflect.Int16, reflect.Int8)
}

func (*standardFormatter) formatBigInt(val interface{}) (string, error) {
	return convert(val, "%d", reflect.Int64, reflect.Int, reflect.Int32,
		reflect.Int16, reflect.Int8)
}

func (*standardFormatter) formatFloat(val interface{}) (string, error) {
	return convert(val, "%f", reflect.Float32)
}

func (*standardFormatter) formatDouble(val interface{}) (string, error) {
	return convert(val, "%f", reflect.Float64, reflect.Float32)
}

func (*standardFormatter) formatVarChar(val interface{}) (string, error) {
	return convert(val, "'%s'", reflect.String)
}

func (*standardFormatter) formatText(val interface{}) (string, error) {
	return convert(val, "'%s'", reflect.String)
}

func (*standardFormatter) formatDate(val interface{}) (string, error) {
	return parseTime(val, "2006-01-02", true)
}

func (*standardFormatter) formatTimeStamp(val interface{}) (string, error) {
	return parseTime(val, "2006-01-02 15:04:05.999999999", true)
}

func (*standardFormatter) formatTimeStampZ(val interface{}) (string, error) {
	return parseTime(val, "2006-01-02 15:04:05.999999999Z07:00", false)
}

func (*standardFormatter) formatBinary(val interface{}) (string, error) {
	typ := reflect.TypeOf(val)
	if typ.AssignableTo(reflect.TypeOf(0)) {
		return fmt.Sprintf("X'%X'", val), nil
	}

	kind := typ.Kind()
	if kind != reflect.Slice && typ.Elem().Kind() != reflect.Uint8 {
		return wrongType(val, "[]byte")
	}

	return fmt.Sprintf("X'%X'", val), nil
}

func (*standardFormatter) formatBlob(val interface{}) (string, error) {
	typ := reflect.TypeOf(val)
	kind := reflect.TypeOf(val).Kind()

	if kind != reflect.Slice && typ.Elem().Kind() != reflect.Uint8 {
		return wrongType(val, "[]byte")
	}

	return fmt.Sprintf("X'%X'", val), nil
}
