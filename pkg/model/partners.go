package model

import (
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
)

func init() {
	database.Tables = append(database.Tables, &Partner{})
}

// Partner represents one record of the 'partners' table.
type Partner struct {
	// The partner's unique ID
	ID uint64 `xorm:"pk autoincr 'id'" json:"id"`
	// The partner's name
	Name string `xorm:"unique(part) notnull 'name'" json:"name"`
	// The ID of the interface this partner is attached to.
	InterfaceID uint64 `xorm:"unique(part) notnull 'interface_id'" json:"interfaceID"`
	// The partner's address
	Address string `xorm:"notnull 'address'" json:"address"`
	// The partner's password
	Port uint16 `xorm:"notnull 'port'" json:"port"`
}

// Validate checks that the partner entry can be inserted into the database
func (p *Partner) Validate(db *database.Db, isInsert bool) error {
	if p.Name == "" {
		return ErrInvalid{msg: "The partner's name cannot be empty"}
	}
	if p.Address == "" {
		return ErrInvalid{msg: "The partner's address cannot be empty"}
	}

	ints, err := db.Query("SELECT id FROM interfaces WHERE id=?", p.InterfaceID)
	if err != nil {
		return err
	}
	if len(ints) == 0 {
		return ErrInvalid{msg: fmt.Sprintf("No interface found with id '%v'", p.InterfaceID)}
	}

	names, err := db.Query("SELECT id FROM partners WHERE interface_id=? AND name=?",
		p.InterfaceID, p.Name)
	if err != nil {
		return err
	}

	if isInsert {
		if len(names) > 0 {
			return ErrInvalid{msg: "A partner with the same name already exist for this interface"}
		}

		ids, err := db.Query("SELECT id FROM partners WHERE id=?", p.ID)
		if err != nil {
			return err
		}
		if len(ids) > 0 {
			return ErrInvalid{msg: "A partner with the same ID already exist"}
		}
	} else {
		if len(names) > 0 && names[0]["id"] != p.ID {
			return ErrInvalid{msg: "A partner with the same name already exist for this interface"}
		}

		res, err := db.Query("SELECT id FROM partners WHERE id=?", p.ID)
		if err != nil {
			return err
		}
		if len(res) == 0 {
			return ErrInvalid{fmt.Sprintf("Unknown partner ID: '%v'", p.ID)}
		}
	}

	return nil
}

// TableName returns the name of the partners SQL table
func (*Partner) TableName() string {
	return "partners"
}
