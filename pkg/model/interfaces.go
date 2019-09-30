package model

import (
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
)

func init() {
	database.Tables = append(database.Tables, &Interface{})
}

// Types is the list of the valid types for an interface.
var Types = []string{"http", "r66", "sftp"}

// Exists returns whether the array of strings 'array' contains the given
// string 'x'.
func Exists(array []string, x string) bool {
	for _, s := range array {
		if s == x {
			return true
		}
	}
	return false
}

// Interface represents a record of the 'interfaces' table.
type Interface struct {
	// The interface's unique ID
	ID uint64 `xorm:"pk autoincr 'id'" json:"id"`
	// The interface's name
	Name string `xorm:"unique notnull 'name'" json:"name"`
	// The interface's used protocol
	Type string `xorm:"notnull 'type'" json:"type"`
	// The port used by the interface
	Port uint16 `xorm:"notnull 'port'" json:"port"`
}

// Validate checks that the interface entry can be inserted into the database
func (i Interface) Validate(db *database.Db, isInsert bool) error {
	if i.Name == "" {
		return ErrInvalid{msg: "The interface's name cannot be empty"}
	}
	if !Exists(Types, i.Type) {
		return ErrInvalid{msg: fmt.Sprintf("The interface's type must be one of %s", Types)}
	}

	if isInsert {
		res, err := db.Query("SELECT id FROM interfaces WHERE id=? OR name=?",
			i.ID, i.Name)
		if err != nil {
			return err
		}
		if len(res) > 0 {
			return ErrInvalid{msg: "An interface with the same ID or name already exist"}
		}
	} else {
		res, err := db.Query("SELECT id FROM interfaces WHERE id=?", i.ID)
		if err != nil {
			return err
		}
		if len(res) == 0 {
			return ErrInvalid{fmt.Sprintf("Unknown interface ID: '%v'", i.ID)}
		}

		nameCheck := &Interface{Name: i.Name}
		if err := db.Get(nameCheck); err != nil {
			if err != database.ErrNotFound {
				return err
			}
		} else if nameCheck.ID != i.ID {
			return ErrInvalid{msg: "An interface with the same name already exist"}
		}
	}

	return nil
}

// TableName returns the name of the partners SQL table
func (Interface) TableName() string {
	return "interfaces"
}
