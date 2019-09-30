package model

import (
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
)

func init() {
	database.Tables = append(database.Tables, &CertChain{})
}

// CertChain represents one record of the 'certificates' table.
type CertChain struct {
	// The account's unique ID
	ID uint64 `xorm:"pk autoincr 'id'" json:"id"`
	// If this certificate is linked to an Account or a Partner
	OwnerType string `xorm:"notnull 'owner_type'" json:"ownerType"`
	// The Id of the account this certificate belongs to
	OwnerID uint64 `xorm:"unique(cert) notnull 'owner_id'" json:"ownerID"`
	// The certificate chain's name
	Name string `xorm:"unique(cert) notnull 'name'" json:"name"`
	// The account's private key
	PrivateKey []byte `xorm:"notnull 'private_key'" json:"privateKey"`
	// The account's public key
	PublicKey []byte `xorm:"notnull 'public_key'" json:"publicKey"`
	// The public key certificate
	Cert []byte `xorm:"notnull 'cert'" json:"cert"`
}

// Validate checks if the certificate entry can be inserted in the database.
func (c *CertChain) Validate(db *database.Db, isInsert bool) error {
	if c.Name == "" {
		return ErrInvalid{msg: "The certificate's name cannot be empty"}
	}
	if c.OwnerID == 0 {
		return ErrInvalid{msg: "The certificate's ownerID cannot be empty"}
	}
	if c.PrivateKey == nil || len(c.PrivateKey) == 0 {
		return ErrInvalid{msg: "The certificate's private key cannot be empty"}
	}
	if c.PublicKey == nil || len(c.PublicKey) == 0 {
		return ErrInvalid{msg: "The certificate's public key cannot be empty"}
	}
	if c.Cert == nil || len(c.Cert) == 0 {
		return ErrInvalid{msg: "The certificate cannot be empty"}
	}
	if c.OwnerType == "ACCOUNT" {
		owner, err := db.Query("SELECT id FROM accounts WHERE id=?", c.OwnerID)
		if err != nil {
			return err
		}
		if len(owner) == 0 {
			return ErrInvalid{msg: fmt.Sprintf("No account found with id '%v'", c.OwnerID)}
		}
	} else if c.OwnerType == "PARTNER" {
		owner, err := db.Query("SELECT id FROM partners WHERE id=?", c.OwnerID)
		if err != nil {
			return err
		}
		if len(owner) == 0 {
			return ErrInvalid{msg: fmt.Sprintf("No partners found with id '%v'", c.OwnerID)}
		}
	} else {
		return ErrInvalid{msg: fmt.Sprintf("Owner type unknown '%v'", c.OwnerType)}
	}

	names, err := db.Query("SELECT id FROM certificates WHERE owner_type =? AND owner_id =? AND name=?",
		c.OwnerType, c.OwnerID, c.Name)
	if err != nil {
		return err
	}

	if isInsert {
		if len(names) > 0 {
			return ErrInvalid{msg: "A certificate with the same name already exist for this account"}
		}

		ids, err := db.Query("SELECT id FROM certificates WHERE id=?", c.ID)
		if err != nil {
			return err
		}
		if len(ids) > 0 {
			return ErrInvalid{msg: "A certificate with the same ID already exist"}
		}
	} else {
		if len(names) > 0 && names[0]["id"] != c.ID {
			return ErrInvalid{msg: "A certificate with the same name already exist for this account"}
		}

		res, err := db.Query("SELECT id FROM certificates WHERE id=?", c.ID)
		if err != nil {
			return err
		}
		if len(res) == 0 {
			return ErrInvalid{fmt.Sprintf("Unknown certificate id: '%v'", c.ID)}
		}
	}
	return nil
}

// TableName returns the name of the certificates SQL table
func (*CertChain) TableName() string {
	return "certificates"
}
