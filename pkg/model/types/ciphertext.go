package types

import (
	"database/sql/driver"
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
)

// CypherText is a wrapper of string which add transparent encryption/decryption
// of the string when storing and loading the string from a database.
type CypherText string

// FromDB takes an slice containing AES encrypted data, and stores the decrypted
// string in the CypherText receiver.
func (c *CypherText) FromDB(bytes []byte) error {
	if len(bytes) == 0 {
		return nil
	}
	plain, err := utils.AESDecrypt(string(bytes))
	if err != nil {
		return err
	}
	*c = CypherText(plain)
	return nil
}

// ToDB takes the string contained in the CypherText receiver, encrypts it using
// AES, and returns the result as a slice of byte.
func (c *CypherText) ToDB() ([]byte, error) {
	if *c == "" {
		return nil, nil
	}
	cypher, err := utils.AESCrypt(string(*c))
	if err != nil {
		return nil, err
	}
	return []byte(cypher), nil
}

// Scan implements database/sql.Scanner. It takes an AES encrypted string and
// returns the
func (c *CypherText) Scan(v interface{}) error {
	switch val := v.(type) {
	case []byte:
		return c.FromDB(val)
	case string:
		return c.FromDB([]byte(val))
	default:
		return fmt.Errorf("type %T is incompatible with CypherText", val)
	}
}

// Value is the equivalent of ToDB for the driver.Valuer interface.
func (c *CypherText) Value() (driver.Value, error) {
	return c.ToDB()
}
