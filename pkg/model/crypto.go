package model

import (
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
	"golang.org/x/crypto/ssh"
)

func init() {
	database.AddTable(&Crypto{})
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

	return c.checkContent(db, parent)
}

func (c *Crypto) checkContentLocal(parent database.GetBean) database.Error {
	if c.PrivateKey == "" {
		return database.NewValidationError("the %s is missing a private key",
			parent.Appellation())
	}
	if c.SSHPublicKey != "" {
		return database.NewValidationError("a %s does not need an SSH public key",
			parent.Appellation())
	}
	return nil
}

func (c *Crypto) checkContentRemote(parent database.GetBean) database.Error {
	if c.Certificate == "" && c.SSHPublicKey == "" {
		return database.NewValidationError(
			"the %s is missing a TLS certificate or an SSH public key",
			parent.Appellation())
	}
	if c.PrivateKey != "" {
		return database.NewValidationError("a %s does not need a private key",
			parent.Appellation())
	}
	return nil
}

func (c *Crypto) checkContent(db database.ReadAccess, parent database.GetBean) database.Error {
	var host, proto string
	var isServer bool
	switch t := parent.(type) {
	case *LocalAgent:
		isServer = true
		host = t.Address
		proto = t.Protocol
		if err := c.checkContentLocal(parent); err != nil {
			return err
		}
	case *LocalAccount:
		host = t.Login
		if err := c.checkContentRemote(parent); err != nil {
			return err
		}
		var parentParent LocalAgent
		if err := db.Get(&parentParent, "id=?", t.LocalAgentID).Run(); err != nil {
			return err
		}
		proto = parentParent.Protocol
	case *RemoteAgent:
		isServer = true
		host = t.Address
		proto = t.Protocol
		if err := c.checkContentRemote(parent); err != nil {
			return err
		}
	case *RemoteAccount:
		host = t.Login
		if err := c.checkContentLocal(parent); err != nil {
			return err
		}
		var parentParent RemoteAgent
		if err := db.Get(&parentParent, "id=?", t.RemoteAgentID).Run(); err != nil {
			return err
		}
		proto = parentParent.Protocol
	}
	return c.validateContent(host, proto, isServer)
}

func (c *Crypto) validateContent(host, proto string, isServer bool) database.Error {
	newErr := database.NewValidationError

	if c.Certificate != "" {
		certChain, err := utils.ParsePEMCertChain(c.Certificate)
		if err != nil {
			return newErr("failed to parse certificate: %s", err)
		}
		if err := utils.CheckCertChain(certChain, isServer); err != nil {
			return newErr("certificate validation failed: %s", err)
		}
		if host != "" && proto != "r66" {
			if err := certChain[0].VerifyHostname(host); err != nil {
				return newErr("the certificate is not valid for host '%s'", host)
			}
		}
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
