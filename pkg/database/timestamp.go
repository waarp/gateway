package database

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"gorm.io/gorm/schema"
)

type timestamp struct{}

//nolint:gochecknoinits //init is needed here
func init() {
	schema.RegisterSerializer("timestamp", timestamp{})
}

func (timestamp) Scan(_ context.Context, field *schema.Field, dst reflect.Value, dbValue any) error {
	switch fieldValue := dbValue.(type) {
	case time.Time:
		if fieldValue.IsZero() {
			return nil
		}

		for dst.Kind() == reflect.Pointer {
			dst = dst.Elem()
		}
		fieldValue = fieldValue.Local()

		dst.FieldByName(field.Name).Set(reflect.ValueOf(fieldValue))
	case nil:
	default:
		//nolint:err113 //too specific to have a base error
		return fmt.Errorf("invalid timestamp type %T", fieldValue)
	}

	return nil
}

//nolint:nilnil // returning "nil nil" is required here to store zero timestamps as "NULL" in db
func (timestamp) Value(_ context.Context, _ *schema.Field, _ reflect.Value, fieldValue any) (any, error) {
	switch val := fieldValue.(type) {
	case time.Time:
		if val.IsZero() {
			return nil, nil
		}

		return val.UTC(), nil
	case nil:
		return nil, nil
	default:
		//nolint:err113 //too specific to have a base error
		return nil, fmt.Errorf("invalid timestamp type %T", fieldValue)
	}
}
