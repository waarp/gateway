package model

import (
	"fmt"
	"sort"
)

func init() {
	Tables = append(Tables, &Partner{})
}

var types = []string{"http", "r66", "sftp"}

// Partner represents one record of the 'partners' table.
type Partner struct {
	// The partner's unique ID
	ID uint64 `xorm:"pk autoincr 'id'" json:"id"`
	// The partner's name
	Name string `xorm:"unique notnull 'name'" json:"name"`
	// The partner's address
	Address string `xorm:"notnull 'address'" json:"address"`
	// The partner's password
	Port uint16 `xorm:"notnull 'port'" json:"port"`
	// The protocol used by the partner
	Type string `xorm:"notnull 'type'" json:"type"`
}

// Validate checks that the partner entry can be inserted into the database
func (p *Partner) Validate(exists func(interface{}) (bool, error)) error {
	if p.Name == "" {
		return ErrInvalid{msg: "The partner's name cannot be empty"}
	}
	if p.Address == "" {
		return ErrInvalid{msg: "The partner's address cannot be empty"}
	}
	if sort.SearchStrings(types, p.Type) == len(types) {
		return ErrInvalid{msg: fmt.Sprintf("The partner's type must be one of %s", types)}
	}

	test := &Partner{Name: p.Name}
	if ok, err := exists(test); err != nil {
		return err
	} else if ok {
		return ErrInvalid{msg: fmt.Sprintf("A partner named '%s' already exists",
			p.Name)}
	}

	return nil
}

// TableName returns the name of the partners SQL table
func (*Partner) TableName() string {
	return "partners"
}
