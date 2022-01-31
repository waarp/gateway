package model

import (
	"database/sql/driver"
	"encoding/binary"
	"fmt"
	"math"
)

// PermsMask is a bitmask specifying which actions the user is allowed to
// perform on the database.
type PermsMask uint32

// Masks for user permissions.
const (
	PermTransfersRead PermsMask = 1 << (32 - 1 - iota)
	PermTransfersWrite
	permTransferDelete // placeholder, transfers CANNOT be deleted by users
	PermServersRead
	PermServersWrite
	PermServersDelete
	PermPartnersRead
	PermPartnersWrite
	PermPartnersDelete
	PermRulesRead
	PermRulesWrite
	PermRulesDelete
	PermUsersRead
	PermUsersWrite
	PermUsersDelete
	PermAdminRead
	PermAdminWrite
	PermAdminDelete

	PermAll PermsMask = math.MaxUint32 &^ permTransferDelete
)

const permMaskSize = 4

// FromDB implements xorm/core.Conversion. As XORM ignores standard converters
// for non-struct types (Value() and Scan()), thus must be mapped to XORM own
// conversion interface.
func (p *PermsMask) FromDB(bytes []byte) error {
	return p.Scan(bytes)
}

// ToDB implements xorm/core.Conversion. As XORM ignores standard converters
// for non-struct types (Value() and Scan()), thus must be mapped to XORM own
// conversion interface.
func (p PermsMask) ToDB() ([]byte, error) {
	v, err := p.Value()

	//nolint:forcetypeassert //no need, the type assertion will always succeed
	return v.([]byte), err
}

// Scan implements database/sql.Scanner. It takes a binary blob and returns
// the matching PermsMask.
func (p *PermsMask) Scan(src interface{}) error {
	switch v := src.(type) {
	case []byte:
		*p = PermsMask(binary.LittleEndian.Uint32(v))

		return nil
	default:
		//nolint:goerr113 // too specific to have a base error
		return fmt.Errorf("cannot scan %+v of type %T into a PermMask", v, v)
	}
}

// Value implements database/driver.Valuer. PermsMask is represented in the
// database as a binary blob.
func (p PermsMask) Value() (driver.Value, error) {
	b := make([]byte, permMaskSize)

	binary.LittleEndian.PutUint32(b, uint32(p))

	return b, nil
}
