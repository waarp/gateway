package model

import (
	"strings"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
	"golang.org/x/crypto/ssh"
)

func init() {
	database.Tables = append(database.Tables, &Crypto{})
}

var validOwnerTypes = []string{TableLocAgents, TableRemAgents, TableLocAccounts,
	TableRemAccounts}

// Crypto represents credentials used to establish secure (encrypted) transfer
// channels. This includes both TLS and SSH tunnels. These credentials can be
// attached to both local/remote agents & local/remote accounts.
type Crypto struct {

	// The credentials' database ID.
	ID uint64 `xorm:"pk autoincr <- 'id'"`

	// The table name of object the credentials are linked to. Valid tables
	// include: local_agent, remote_agent, local_account or remote_account.
	OwnerType string `xorm:"unique(cert) notnull 'owner_type'"`

	// The id of the object these credentials are linked to.
	OwnerID uint64 `xorm:"unique(cert) notnull 'owner_id'"`

	// The credentials' display name.
	Name string `xorm:"unique(cert) notnull 'name'"`

	// A PEM encoded TLS private key.
	PrivateKey types.CypherText `xorm:"text 'private_key'"`

	// A PEM encoded TLS certificate.
	Certificate string `xorm:"text 'certificate'"`

	// An SSH public key in authorized_keys format.
	SSHPublicKey string `xorm:"text 'ssh_public_key'"`

	// A base64 representation of the sha256 checksum of the raw certificate
	// (if one is provided), used for indexation & search purposes.
	Signature string `xorm:"varchar(44) unique 'signature'"`
}

// TableName returns the name of the certificates table.
func (*Crypto) TableName() string {
	return TableCrypto
}

// Appellation returns the name of 1 element of the certificates table.
func (*Crypto) Appellation() string {
	return "crypto credentials"
}

// GetID returns the certificate's ID.
func (c *Crypto) GetID() uint64 {
	return c.ID
}

// BeforeWrite checks if the new `Crypto` entry is valid and can be inserted
// in the database.
func (c *Crypto) BeforeWrite(db database.ReadAccess) database.Error {
	newErr := database.NewValidationError

	if c.Name == "" {
		return newErr("the credentials' name cannot be empty")
	}
	if c.OwnerType == "" {
		return newErr("the credentials' owner type is missing")
	}
	if c.OwnerID == 0 {
		return newErr("the credentials' owner ID is missing")
	}

	if c.Certificate != "" && c.SSHPublicKey != "" {
		return newErr("secure credentials should not contain both " +
			"a certificate and an SSH public key (both cannot be used at the same time)")
	}

	n, err := db.Count(c).Where("id<>? AND owner_type=? AND owner_id=? AND name=?",
		c.ID, c.OwnerType, c.OwnerID, c.Name).Run()
	if err != nil {
		return err
	} else if n > 0 {
		return newErr("credentials with the same name '%s' already exist", c.Name)
	}

	var parent database.GetBean
	switch c.OwnerType {
	case TableLocAgents:
		parent = &LocalAgent{}
	case TableRemAgents:
		parent = &RemoteAgent{}
	case TableLocAccounts:
		parent = &LocalAccount{}
	case TableRemAccounts:
		parent = &RemoteAccount{}
	default:
		return newErr("the credentials' owner type must be one of %s", validOwnerTypes)
	}
	if err := db.Get(parent, "id=?", c.OwnerID).Run(); err != nil {
		if database.IsNotFound(err) {
			return newErr("no %s found with ID '%v'", parent.Appellation(), c.OwnerID)
		}
		return err
	}

	return c.checkContent(parent)
}

func (c *Crypto) checkContent(parent database.GetBean) database.Error {
	newErr := database.NewValidationError

	var addr string
	if c.OwnerType == TableLocAgents || c.OwnerType == TableRemAccounts {
		if t, ok := parent.(*LocalAgent); ok {
			addr = t.Address
		}
		if c.PrivateKey == "" {
			return newErr("the %s is missing a private key", parent.Appellation())
		}
		if c.SSHPublicKey != "" {
			return newErr("a %s does not need an SSH public key", parent.Appellation())
		}
	} else {
		if t, ok := parent.(*RemoteAgent); ok {
			addr = t.Address
		}
		if c.Certificate == "" && c.SSHPublicKey == "" {
			return newErr("the %s is missing a TLS certificate or an SSH public key",
				parent.Appellation())
		}
		if c.PrivateKey != "" {
			return newErr("a %s does not need a private key", parent.Appellation())
		}
	}

	return c.validateContent(addr)
}

func (c *Crypto) validateContent(addr string) database.Error {
	newErr := database.NewValidationError

	if c.Certificate != "" {
		certChain, err := utils.ParsePEMCertChain(c.Certificate)
		if err != nil {
			return newErr("failed to parse certificate: %s", err)
		}
		if err := utils.CheckCertChain(certChain, addr); err != nil {
			return newErr("certificate validation failed: %s", err)
		}
		c.Signature = utils.MakeSignature(certChain[0])
	}

	if c.PrivateKey != "" {
		if _, err := ssh.ParsePrivateKey([]byte(c.PrivateKey)); err != nil {
			return newErr("failed to parse private key: %s",
				strings.TrimPrefix(err.Error(), "ssh: "))
		}
	}

	if c.SSHPublicKey != "" {
		if _, _, _, _, err := ssh.ParseAuthorizedKey([]byte(c.SSHPublicKey)); err != nil { //nolint:dogsled
			return newErr("failed to parse SSH public key: %s",
				strings.TrimPrefix(err.Error(), "ssh: "))
		}
	}

	return nil
}
