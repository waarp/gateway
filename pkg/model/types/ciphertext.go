package types

import (
	"database/sql/driver"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

// SecretText is a wrapper of string which add transparent encryption/decryption
// of the string when storing and loading the string from a database.
type SecretText string

func (s *SecretText) String() string { return string(*s) }

// FromDB takes a slice containing AES encrypted data, and stores the decrypted
// string in the SecretText receiver.
func (s *SecretText) FromDB(bytes []byte) error {
	if len(bytes) == 0 {
		return nil
	}

	plain, err := utils.AESDecrypt(database.GCM, string(bytes))
	if err != nil {
		return fmt.Errorf("cannot decrypt password: %w", err)
	}

	*s = SecretText(plain)

	return nil
}

// ToDB takes the string contained in the SecretText receiver, encrypts it using
// AES, and returns the result as a slice of byte.
func (s *SecretText) ToDB() ([]byte, error) {
	if *s == "" {
		return []byte(""), nil
	}

	cypher, err := utils.AESCrypt(database.GCM, string(*s))
	if err != nil {
		return nil, fmt.Errorf("cannot encrypt password: %w", err)
	}

	return []byte(cypher), nil
}

// Scan implements database/sql.Scanner. It takes an AES encrypted string and
// sets the object.
func (s *SecretText) Scan(v interface{}) error {
	switch val := v.(type) {
	case []byte:
		return s.FromDB(val)
	case string:
		return s.FromDB([]byte(val))
	default:
		//nolint:goerr113 // too specific to have a base error
		return fmt.Errorf("type %T is incompatible with SecretText", val)
	}
}

// Value is the equivalent of ToDB for the driver.Valuer interface.
func (s *SecretText) Value() (driver.Value, error) {
	return s.ToDB()
}
