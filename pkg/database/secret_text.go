package database

import (
	"context"
	"crypto/cipher"
	"errors"
	"fmt"
	"reflect"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type aeadKeyType string

const aeadKey aeadKeyType = "aead"

var errNoAEAD = errors.New("no aead key found in context")

//nolint:gochecknoinits //init is needed here
func init() {
	schema.RegisterSerializer("secret", secretText{})
}

func registerAEAD(db *gorm.DB, aead cipher.AEAD) {
	ctx := context.WithValue(context.Background(), aeadKey, aead)
	*db = *db.WithContext(ctx)
}

// secretText is a wrapper of string which add transparent encryption/decryption
// of the string when storing and loading the string from a database.
type secretText struct{}

func (secretText) Scan(ctx context.Context, field *schema.Field, dst reflect.Value, dbValue any) error {
	key, ok := ctx.Value(aeadKey).(cipher.AEAD)
	if !ok {
		return errNoAEAD
	}

	var (
		cipherText string
		err        error
	)

	switch val := dbValue.(type) {
	case string:
		cipherText, err = AESDecrypt(key, val)
	case []byte:
		cipherText, err = AESDecrypt(key, string(val))
	default:
		//nolint:err113 //too specific to have a base error
		return fmt.Errorf("unsupported type for SecretText: %T", dbValue)
	}
	if err != nil {
		return fmt.Errorf("failed to encrypt secret text: %w", err)
	}

	for dst.Kind() == reflect.Pointer {
		dst = dst.Elem()
	}

	dst.FieldByName(field.Name).SetString(cipherText)

	return nil
}

func (secretText) Value(ctx context.Context, _ *schema.Field, _ reflect.Value, fieldValue any) (any, error) {
	key, ok := ctx.Value(aeadKey).(cipher.AEAD)
	if !ok {
		return nil, errNoAEAD
	}

	switch val := fieldValue.(type) {
	case string:
		return AESEncrypt(key, val)
	case []byte:
		return AESEncrypt(key, string(val))
	default:
		//nolint:err113 //too specific to have a base error
		return nil, fmt.Errorf("unsupported type for SecretText: %T", fieldValue)
	}
}
