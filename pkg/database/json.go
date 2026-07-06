package database

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	"gorm.io/gorm/schema"
)

type jsonSerializer struct {
	schema.JSONSerializer
}

//nolint:gochecknoinits //init is needed here
func init() {
	schema.RegisterSerializer("json", jsonSerializer{})
}

func (jsonSerializer) Scan(ctx context.Context, field *schema.Field, dst reflect.Value, dbValue any) error {
	fieldValue := reflect.New(field.FieldType)
	var r io.Reader

	if dbValue != nil {
		switch v := dbValue.(type) {
		case []byte:
			r = bytes.NewReader(v)
		case string:
			r = strings.NewReader(v)
		default:
			//nolint:err113 //too specific to have a base error
			return fmt.Errorf("invalid json type %T", dbValue)
		}
	}

	decoder := json.NewDecoder(r)
	decoder.UseNumber()
	if err := decoder.Decode(fieldValue.Interface()); err != nil {
		return fmt.Errorf("failed to unmarshal json value: %w", err)
	}

	field.ReflectValueOf(ctx, dst).Set(fieldValue.Elem())

	return nil
}
