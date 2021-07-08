package model

import (
	"fmt"
)

// Signature is a alias for a byte array containing a sha256 checksum.
type Signature [32]byte

// FromDB implements xorm/core.Conversion. As XORM ignores standard converters
// for non-struct types (Value() and Scan()), thus must be mapped to XORM own
// conversion interface.
func (s *Signature) FromDB(bytes []byte) error {
	return s.Scan(bytes)
}

// Scan implements database/sql.Scanner. It takes a binary blob and returns
// the matching Signature.
func (s *Signature) Scan(src interface{}) error {
	switch v := src.(type) {
	case []byte:
		copy(s[:], v)
		return nil
	default:
		return fmt.Errorf("cannot scan %+v of type %T into a Signature", v, v)
	}
}
