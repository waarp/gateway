package model

import (
	"database/sql/driver"
	"encoding/binary"
	"fmt"
	"math"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
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

	PermAll = math.MaxUint32 &^ permTransferDelete
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

// Permissions is a structured representation of a PermMask which regroups
// permissions into categories depending on their target. Each attribute
// represents 1 target. The attributes are strings which give a chmod-like
// representation of the permission.
type Permissions struct {
	Transfers      string
	Servers        string
	Partners       string
	Rules          string
	Users          string
	Administration string
}

const permString = "rwd"

func maskToStr(m PermsMask, s int) string {
	buf := make([]byte, len(permString))

	for i, c := range permString {
		if m&(1<<uint(32-1-s-i)) != 0 {
			buf[i] = byte(c)
		} else {
			buf[i] = '-'
		}
	}

	return string(buf)
}

// MaskToPerms converts a PermMask to an equivalent Permissions instance.
func MaskToPerms(m PermsMask) *Permissions {
	//nolint:gomnd //too specific
	return &Permissions{
		Transfers:      maskToStr(m, 0*len(permString)),
		Servers:        maskToStr(m, 1*len(permString)),
		Partners:       maskToStr(m, 2*len(permString)),
		Rules:          maskToStr(m, 3*len(permString)),
		Users:          maskToStr(m, 4*len(permString)),
		Administration: maskToStr(m, 5*len(permString)),
	}
}

func permToMask(mask *PermsMask, perm string, off int) database.Error {
	invalid := func(format string, args ...interface{}) database.Error {
		reason := fmt.Sprintf(format, args...)

		return database.NewValidationError("invalid permission string '%s': %s", perm, reason)
	}

	if len(perm) == 0 {
		return nil
	}

	if len(perm) != len(permString) {
		return invalid("expected length 3, got %d", len(perm))
	}

	process := func(o int, expected rune) database.Error {
		switch char := rune(perm[o]); char {
		case '-':
		case expected:
			*mask |= 1 << (32 - 1 - off - o)
		default:
			return invalid("invalid permission mode '%c' (expected '%c' or '-')", char, expected)
		}

		return nil
	}

	for o, r := range permString {
		if err := process(o, r); err != nil {
			return err
		}
	}

	return nil
}

// PermsToMask converts the given Permissions instance to an equivalent PermsMask.
//nolint:gomnd //too specific
func PermsToMask(perms *Permissions) (PermsMask, database.Error) {
	if perms == nil {
		return 0, nil
	}

	var mask PermsMask
	if err := permToMask(&mask, perms.Transfers, 0*len(permString)); err != nil {
		return 0, err
	}

	if err := permToMask(&mask, perms.Servers, 1*len(permString)); err != nil {
		return 0, err
	}

	if err := permToMask(&mask, perms.Partners, 2*len(permString)); err != nil {
		return 0, err
	}

	if err := permToMask(&mask, perms.Rules, 3*len(permString)); err != nil {
		return 0, err
	}

	if err := permToMask(&mask, perms.Users, 4*len(permString)); err != nil {
		return 0, err
	}

	if err := permToMask(&mask, perms.Administration, 5*len(permString)); err != nil {
		return 0, err
	}

	return mask, nil
}
