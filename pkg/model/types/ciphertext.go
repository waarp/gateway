package types

import (
	"database/sql/driver"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

// CypherText is a wrapper of string which add transparent encryption/decryption
// of the string when storing and loading the string from a database.
type CypherText string

func (c *CypherText) String() string { return string(*c) }

// FromDB takes an slice containing AES encrypted data, and stores the decrypted
// string in the CypherText receiver.
func (c *CypherText) FromDB(bytes []byte) error {
	if len(bytes) == 0 {
		return nil
	}

	plain, err := utils.AESDecrypt(database.GCM, string(bytes))
	if err != nil {
		return fmt.Errorf("cannot decrypt password: %w", err)
	}

	*c = CypherText(plain)

	return nil
}

// ToDB takes the string contained in the CypherText receiver, encrypts it using
// AES, and returns the result as a slice of byte.
func (c *CypherText) ToDB() ([]byte, error) {
	if *c == "" {
		return []byte(""), nil
	}

	cypher, err := utils.AESCrypt(database.GCM, string(*c))
	if err != nil {
		return nil, fmt.Errorf("cannot encrypt password: %w", err)
	}

	return []byte(cypher), nil
}

// Scan implements database/sql.Scanner. It takes an AES encrypted string and
// sets the object.
func (c *CypherText) Scan(v interface{}) error {
	switch val := v.(type) {
	case []byte:
		return c.FromDB(val)
	case string:
		return c.FromDB([]byte(val))
	default:
		//nolint:goerr113 // too specific to have a base error
		return fmt.Errorf("type %T is incompatible with CypherText", val)
	}
}

// Value is the equivalent of ToDB for the driver.Valuer interface.
func (c *CypherText) Value() (driver.Value, error) {
	return c.ToDB()
}
