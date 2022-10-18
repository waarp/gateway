package model

import (
	"database/sql"
	"fmt"
	"net"
	"strings"

	"golang.org/x/crypto/ssh"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

//nolint:gochecknoinits // init is used by design
func init() {
	database.AddTable(&Crypto{})
}

// CryptoOwner is the interface implemented by all valid Crypto owner types.
// Valid owner types are LocalAgent, RemoteAgent, LocalAccount & RemoteAccount.
type CryptoOwner interface {
	// SetCryptoOwner sets the target CryptoOwner as owner of the given Crypto
	// instance (by setting the corresponding foreign key to its own ID).
	SetCryptoOwner(*Crypto)

	// GenCryptoSelectCond the SQL "WHERE" condition for selecting the Crypto
	// entries belonging to this owner.
	GenCryptoSelectCond() (string, int64)
}

// Crypto represents credentials used to establish secure (encrypted) transfer
// channels. This includes both TLS and SSH tunnels. These credentials can be
// attached to both local/remote agents & local/remote accounts.
type Crypto struct {
	// The credentials' database ID.
	ID int64 `xorm:"PK AUTOINCR <- 'id'"`

	// The id of the object these credentials are linked to.
	LocalAgentID    sql.NullInt64 `xorm:"BIGINT UNIQUE(locAg) 'local_agent_id'"`     // foreign key (local_agents.id)
	RemoteAgentID   sql.NullInt64 `xorm:"BIGINT UNIQUE(remAg) 'remote_agent_id'"`    // foreign key (remote_agents.id)
	LocalAccountID  sql.NullInt64 `xorm:"BIGINT UNIQUE(locAcc) 'local_account_id'"`  // foreign key (local_accounts.id)
	RemoteAccountID sql.NullInt64 `xorm:"BIGINT UNIQUE(remAcc) 'remote_account_id'"` // foreign key (remote_accounts.id)

	// The credentials' display name.
	Name string `xorm:"VARCHAR(100) UNIQUE(locAg) UNIQUE(remAg) UNIQUE(locAcc) UNIQUE(remAcc) NOTNULL 'name'"`

	// A PEM encoded TLS private key.
	PrivateKey types.CypherText `xorm:"TEXT NOTNULL DEFAULT('') 'private_key'"`

	// A PEM encoded TLS certificate.
	Certificate string `xorm:"TEXT NOTNULL DEFAULT('') 'certificate'"`

	// An SSH public key in authorized_keys format.
	SSHPublicKey string `xorm:"TEXT NOTNULL DEFAULT('') 'ssh_public_key'"`
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
func (c *Crypto) GetID() int64 {
	return c.ID
}

// BeforeWrite checks if the new `Crypto` entry is valid and can be inserted
// in the database.
func (c *Crypto) BeforeWrite(db database.ReadAccess) database.Error {
	newErr := database.NewValidationError

	if c.Name == "" {
		return newErr("the credentials' name cannot be empty")
	}

	if c.Certificate != "" && c.SSHPublicKey != "" {
		return newErr("secure credentials should not contain both " +
			"a certificate and an SSH public key (both cannot be used at the same time)")
	}

	if sum := boolToInt(c.LocalAgentID.Valid) + boolToInt(c.RemoteAgentID.Valid) +
		boolToInt(c.LocalAccountID.Valid) + boolToInt(c.RemoteAccountID.Valid); sum == 0 {
		return database.NewValidationError("the crypto credential is missing an owner")
	} else if sum > 1 {
		return database.NewValidationError("the crypto credential cannot have multiple targets")
	}

	var owner interface {
		database.UpdateBean
		CryptoOwner
	}

	switch {
	case c.LocalAgentID.Valid:
		owner = &LocalAgent{ID: c.LocalAgentID.Int64}
	case c.RemoteAgentID.Valid:
		owner = &RemoteAgent{ID: c.RemoteAgentID.Int64}
	case c.LocalAccountID.Valid:
		owner = &LocalAccount{ID: c.LocalAccountID.Int64}
	case c.RemoteAccountID.Valid:
		owner = &RemoteAccount{ID: c.RemoteAccountID.Int64}
	default:
		return database.NewValidationError("the rule access is missing a target") // impossible
	}

	if err := db.Get(owner, "id=?", owner.GetID()).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationError("no %s found with ID '%v'",
				owner.Appellation(), owner.GetID())
		}

		return err
	}

	if n, err := db.Count(c).Where("id<>? AND name=?", c.ID, c.Name).Where(
		owner.GenCryptoSelectCond()).Run(); err != nil {
		return err
	} else if n > 0 {
		return database.NewValidationError("crypto credentials with the same "+
			"name '%s' already exist", c.Name)
	}

	return c.checkContent(db, owner)
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

//nolint:funlen // splitting the function would add complexity
func (c *Crypto) checkContent(db database.ReadAccess, parent database.GetBean) database.Error {
	var (
		host, proto string
		isServer    bool
		err         error
	)

	switch owner := parent.(type) {
	case *LocalAgent:
		isServer = true

		host, _, err = net.SplitHostPort(owner.Address)
		if err != nil {
			return database.NewValidationError("failed to parse certificate owner address")
		}

		proto = owner.Protocol

		if err := c.checkContentLocal(parent); err != nil {
			return err
		}

	case *LocalAccount:
		host = owner.Login

		if err := c.checkContentRemote(parent); err != nil {
			return err
		}

		var parentParent LocalAgent
		if err := db.Get(&parentParent, "id=?", owner.LocalAgentID).Run(); err != nil {
			return err
		}

		proto = parentParent.Protocol

	case *RemoteAgent:
		isServer = true

		host, _, err = net.SplitHostPort(owner.Address)
		if err != nil {
			return database.NewValidationError("failed to parse certificate owner address")
		}

		proto = owner.Protocol

		if err := c.checkContentRemote(parent); err != nil {
			return err
		}

	case *RemoteAccount:
		host = owner.Login

		if err := c.checkContentLocal(parent); err != nil {
			return err
		}

		var parentParent RemoteAgent
		if err := db.Get(&parentParent, "id=?", owner.RemoteAgentID).Run(); err != nil {
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

		if proto != config.ProtocolR66TLS || !isLegacyR66Cert(certChain[0]) {
			if err := utils.CheckCertChain(certChain, isServer, host); err != nil {
				return newErr(err.Error())
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
		if _, err := utils.ParseSSHAuthorizedKey(c.SSHPublicKey); err != nil {
			return newErr("failed to parse SSH public key: %s",
				strings.TrimPrefix(err.Error(), "ssh: "))
		}
	}

	return nil
}

func (*Crypto) MakeExtraConstraints(db *database.Executor) database.Error {
	// add a foreign key to 'local_agent_id'
	if err := redefineColumn(db, TableCrypto, "local_agent_id", fmt.Sprintf(
		`BIGINT REFERENCES %s ON UPDATE RESTRICT ON DELETE CASCADE`,
		TableLocAgents)); err != nil {
		return err
	}

	// add a foreign key to 'remote_agent_id'
	if err := redefineColumn(db, TableCrypto, "remote_agent_id", fmt.Sprintf(
		`BIGINT REFERENCES %s ON UPDATE RESTRICT ON DELETE CASCADE`,
		TableRemAgents)); err != nil {
		return err
	}

	// add a foreign key to 'local_account_id'
	if err := redefineColumn(db, TableCrypto, "local_account_id", fmt.Sprintf(
		`BIGINT REFERENCES %s ON UPDATE RESTRICT ON DELETE CASCADE`,
		TableLocAccounts)); err != nil {
		return err
	}

	// add a foreign key to 'remote_account_id'
	if err := redefineColumn(db, TableCrypto, "remote_account_id", fmt.Sprintf(
		`BIGINT REFERENCES %s ON UPDATE RESTRICT ON DELETE CASCADE`,
		TableRemAccounts)); err != nil {
		return err
	}

	// add a constraint to enforce that one (and ONLY ONE) of 'local_agent_id',
	// 'remote_agent_id', 'local_account_id' and 'remote_account_id' must be defined
	return addTableConstraint(db, TableCrypto,
		`CHECK ( (local_agent_id IS NOT NULL) + (remote_agent_id IS NOT NULL) + `+
			`(local_account_id IS NOT NULL) + (remote_account_id IS NOT NULL) = 1 )`)
}
