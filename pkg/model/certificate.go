package model

import "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"

func init() {
	database.Tables = append(database.Tables, &Cert{})
}

var validOwnerTypes = []string{(&LocalAgent{}).TableName(), (&RemoteAgent{}).TableName(),
	(&LocalAccount{}).TableName(), (&RemoteAccount{}).TableName()}

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
	PublicKey []byte `xorm:"notnull 'public_key'"`

	// The content of the certificate
	Certificate []byte `xorm:"notnull 'cert'"`
}

// TableName returns the name of the certificates table.
func (c *Cert) TableName() string {
	return "certificates"
}

// ValidateInsert checks if the new `Cert` entry is valid and can be inserted
// in the database.
func (c *Cert) ValidateInsert(acc database.Accessor) error {
	if c.ID != 0 {
		return database.InvalidError("The certificate's ID cannot be entered manually")
	}
	if c.OwnerType == "" {
		return database.InvalidError("The certificate's owner type is missing")
	}
	if c.OwnerID == 0 {
		return database.InvalidError("The certificate's owner ID is missing")
	}
	if c.Name == "" {
		return database.InvalidError("The certificate's name cannot be empty")
	}
	if c.OwnerType == (&LocalAgent{}).TableName() && len(c.PrivateKey) == 0 {
		return database.InvalidError("The certificate's private key is missing")
	}
	if len(c.PublicKey) == 0 {
		return database.InvalidError("The certificate's public key is missing")
	}
	if len(c.Certificate) == 0 {
		return database.InvalidError("The certificate's content is missing")
	}

	var res []map[string]interface{}
	var err error
	switch c.OwnerType {
	case "local_agents":
		res, err = acc.Query("SELECT id FROM local_agents WHERE id=?", c.OwnerID)
	case "remote_agents":
		res, err = acc.Query("SELECT id FROM remote_agents WHERE id=?", c.OwnerID)
	case "local_accounts":
		res, err = acc.Query("SELECT id FROM local_accounts WHERE id=?", c.OwnerID)
	case "remote_accounts":
		res, err = acc.Query("SELECT id FROM remote_accounts WHERE id=?", c.OwnerID)
	default:
		return database.InvalidError("The certificate's owner type must be one of %s",
			validOwnerTypes)
	}
	if err != nil {
		return err
	} else if len(res) == 0 {
		return database.InvalidError("No "+c.OwnerType+" found with ID '%v'", c.OwnerID)
	}

	if res, err := acc.Query("SELECT id FROM certificates WHERE owner_type=? AND owner_id=? "+
		"AND name=?", c.OwnerType, c.OwnerID, c.Name); err != nil {
		return err
	} else if len(res) > 0 {
		return database.InvalidError("A certificate with the same name '%s' "+
			"already exist", c.Name)
	}

	return nil
}

// ValidateUpdate checks if the updated `Cert` entry is valid and can be inserted
// in the database.
func (c *Cert) ValidateUpdate(acc database.Accessor, id uint64) error {
	var res []map[string]interface{}
	var err error

	old := Cert{ID: id}
	if err := acc.Get(&old); err != nil {
		return err
	}
	if c.OwnerType != "" {
		old.OwnerType = c.OwnerType
	}
	if c.OwnerID != 0 {
		old.OwnerID = c.OwnerID
	}
	if c.Name != "" {
		old.Name = c.Name
	}

	if c.OwnerType != "" || c.OwnerID != 0 {
		switch c.OwnerType {
		case "local_agents":
			res, err = acc.Query("SELECT id FROM local_agents WHERE id=?", old.OwnerID)
		case "remote_agents":
			res, err = acc.Query("SELECT id FROM remote_agents WHERE id=?", old.OwnerID)
		case "local_accounts":
			res, err = acc.Query("SELECT id FROM local_accounts WHERE id=?", old.OwnerID)
		case "remote_accounts":
			res, err = acc.Query("SELECT id FROM remote_accounts WHERE id=?", old.OwnerID)
		default:
			return database.InvalidError("The certificate's owner type must be one of %s",
				validOwnerTypes)
		}
		if err != nil {
			return err
		} else if len(res) == 0 {
			return database.InvalidError("No "+old.OwnerType+" found with ID '%v'", old.OwnerID)
		}
	}

	if c.Name != "" {
		if res, err := acc.Query("SELECT id FROM certificates WHERE owner_type=? AND owner_id=? "+
			"AND name=?", old.OwnerType, old.OwnerID, old.Name); err != nil {
			return err
		} else if len(res) > 0 {
			return database.InvalidError("A certificate with the same name '%s' "+
				"already exist", c.Name)
		}
	}

	return nil
}
