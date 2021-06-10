package model

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
)

func init() {
	database.Tables = append(database.Tables, &Cert{})
}

var validOwnerTypes = []string{"local_agents", "remote_agents", "local_accounts",
	"remote_accounts"}

// Cert represents a certificate entry, along with its' public and private keys.
// A certificate can be attached to agents and accounts.
type Cert struct {

	// The certificate's database ID
	ID uint64 `xorm:"pk autoincr <- 'id'"`

	// The type of object the certificate is linked to (local_agent, remote_agent,
	// local_account or remote_account)
	OwnerType string `xorm:"unique(cert) notnull 'owner_type'"`

	// The id of the object this certificate is linked to.
	OwnerID uint64 `xorm:"unique(cert) notnull 'owner_id'"`

	// The certificate's name
	Name string `xorm:"unique(cert) notnull 'name'"`

	// The certificate's private key
	PrivateKey []byte `xorm:"'private_key'"`

	// The certificate's public key
	PublicKey []byte `xorm:"'public_key'"`

	// The content of the certificate
	Certificate []byte `xorm:"'cert'"`
}

// TableName returns the name of the certificates table.
func (*Cert) TableName() string {
	return "certificates"
}

// Appellation returns the name of 1 element of the certificates table.
func (*Cert) Appellation() string {
	return "certificate"
}

// GetID returns the certificate's ID.
func (c *Cert) GetID() uint64 {
	return c.ID
}

// BeforeWrite checks if the new `Cert` entry is valid and can be inserted
// in the database.
func (c *Cert) BeforeWrite(db database.ReadAccess) database.Error {
	if c.OwnerType == "" {
		return database.NewValidationError("the certificate's owner type is missing")
	}
	if c.OwnerID == 0 {
		return database.NewValidationError("the certificate's owner ID is missing")
	}
	if c.Name == "" {
		return database.NewValidationError("the certificate's name cannot be empty")
	}
	if (c.OwnerType == "remote_accounts" || c.OwnerType == "local_agents") &&
		len(c.PrivateKey) == 0 {
		return database.NewValidationError("the certificate's private key is missing")
	}
	if (c.OwnerType == "remote_agents" || c.OwnerType == "local_accounts") &&
		len(c.PublicKey) == 0 {
		return database.NewValidationError("the certificate's public key is missing")
	}

	var n uint64
	var err database.Error
	switch c.OwnerType {
	case "local_agents":
		n, err = db.Count(&LocalAgent{}).Where("id=?", c.OwnerID).Run()
	case "remote_agents":
		n, err = db.Count(&RemoteAgent{}).Where("id=?", c.OwnerID).Run()
	case "local_accounts":
		n, err = db.Count(&LocalAccount{}).Where("id=?", c.OwnerID).Run()
	case "remote_accounts":
		n, err = db.Count(&RemoteAccount{}).Where("id=?", c.OwnerID).Run()
	default:
		return database.NewValidationError("the certificate's owner type must be one of %s",
			validOwnerTypes)
	}
	if err != nil {
		return err
	} else if n == 0 {
		return database.NewValidationError("no %s found with ID '%v'", c.OwnerType,
			c.OwnerID)
	}

	n, err = db.Count(c).Where("id<>? AND owner_type=? AND owner_id=? AND name=?",
		c.ID, c.OwnerType, c.OwnerID, c.Name).Run()
	if err != nil {
		return err
	} else if n > 0 {
		return database.NewValidationError(
			"a certificate with the same name '%s' already exist", c.Name)
	}

	return nil
}
