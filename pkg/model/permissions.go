package model

import (
	"fmt"
	"math"
)

// PermsMask is a bitmask specifying which actions the user is allowed to
// perform on the database.
type PermsMask int32

// Masks for user permissions.
const (
	PermTransfersRead PermsMask = 1 << iota
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

	PermAll = math.MaxInt32 &^ permTransferDelete
)

func (m PermsMask) HasPermission(perms ...PermsMask) bool {
	for _, perm := range perms {
		if m&perm == perm {
			return true
		}
	}

	return false
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
		if m&(1<<uint(s+i)) != 0 {
			buf[i] = byte(c)
		} else {
			buf[i] = '-'
		}
	}

	return string(buf)
}

// MaskToPerms converts a PermMask to an equivalent Permissions instance.
func MaskToPerms(m PermsMask) *Permissions {
	//nolint:mnd //too specific
	return &Permissions{
		Transfers:      maskToStr(m, 0*len(permString)),
		Servers:        maskToStr(m, 1*len(permString)),
		Partners:       maskToStr(m, 2*len(permString)),
		Rules:          maskToStr(m, 3*len(permString)),
		Users:          maskToStr(m, 4*len(permString)),
		Administration: maskToStr(m, 5*len(permString)),
	}
}

func permToMask(mask *PermsMask, perm string, off int) error {
	invalid := func(format string, args ...interface{}) error {
		reason := fmt.Sprintf(format, args...)

		//nolint:goerr113 //too specific to have a base error
		return fmt.Errorf("invalid permission string %q: %s", perm, reason)
	}

	if len(perm) == 0 {
		return nil
	}

	if len(perm) != len(permString) {
		return invalid("expected length 3, got %d", len(perm))
	}

	process := func(o int, expected rune) error {
		switch char := rune(perm[o]); char {
		case '-':
		case expected:
			*mask |= 1 << (off + o)
		default:
			return invalid(`invalid permission mode "%c" (expected "%c" or "-")`, char, expected)
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
//
//nolint:mnd //too specific
func PermsToMask(perms *Permissions) (PermsMask, error) {
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
