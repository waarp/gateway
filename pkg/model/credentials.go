package model

import (
	"database/sql"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication"
)

// CredOwnerTable is the interface implemented by all valid Credential owner table types.
// Valid owner types are LocalAgent, RemoteAgent, LocalAccount & RemoteAccount.
type CredOwnerTable interface {
	authentication.Owner
	database.Identifier
	database.Table

	// SetCredOwner sets the target CredOwnerTable as owner of the given Credential
	// instance (by setting the corresponding foreign key to its own ID).
	SetCredOwner(cred *Credential)

	// GetCredentials returns the list of all the owner's authentication methods
	// of the given types. If no type is given, all types are returned.
	GetCredentials(db database.ReadAccess, credTypes ...string) (Credentials, error)
}

// Credential represents a triplet comprised of an authentication method, and two
// authentication value. That triplet is attached to an owner, and can be used to
// authenticate said owner. An owner can have any number of Credential attached
// to it.
type Credential struct {
	ID int64 `xorm:"<- id AUTOINCR"` // The credential's database ID.

	// The id of the object these credentials are linked to. Only one can be
	// valid at a time.
	LocalAgentID    sql.NullInt64 `xorm:"local_agent_id"`
	RemoteAgentID   sql.NullInt64 `xorm:"remote_agent_id"`
	LocalAccountID  sql.NullInt64 `xorm:"local_account_id"`
	RemoteAccountID sql.NullInt64 `xorm:"remote_account_id"`

	Name string `xorm:"name"` // The credentials' display name.
	Type string `xorm:"type"` // The method of authentication.

	// The authentication value (i.e. the password, certificate, hash...) in text format.
	Value string `xorm:"value"`

	// The secondary authentication value (when applicable) in text format.
	Value2 string `xorm:"value2"`
}

func (c *Credential) GetID() int64      { return c.ID }
func (*Credential) TableName() string   { return TableCredentials }
func (*Credential) Appellation() string { return NameCredentials }

// BeforeWrite checks if the new `Crypto` entry is valid and can be inserted
// in the database.
func (c *Credential) BeforeWrite(db database.Access) error {
	if c.Type == "" {
		return database.NewValidationError("the authentication method's type is missing")
	}

	if c.Name == "" {
		c.Name = c.Type
	}

	if c.Value == "" {
		return database.NewValidationError("the authentication value is missing")
	}

	if sum := countTrue(c.LocalAgentID.Valid, c.RemoteAgentID.Valid,
		c.LocalAccountID.Valid, c.RemoteAccountID.Valid); sum == 0 {
		return database.NewValidationError("the authentication method is missing an owner")
	} else if sum > 1 {
		return database.NewValidationError("the authentication method cannot have multiple targets")
	}

	return c.validate(db)
}

func (c *Credential) getOwner(db database.ReadAccess) (CredOwnerTable, authentication.Handler, error) {
	var owner interface {
		database.GetBean
		CredOwnerTable
	}

	var handler authentication.Handler

	switch {
	case c.LocalAgentID.Valid:
		owner = &LocalAgent{ID: c.LocalAgentID.Int64}
		handler = authentication.GetExternalAuthMethod(c.Type)
	case c.RemoteAgentID.Valid:
		owner = &RemoteAgent{ID: c.RemoteAgentID.Int64}
		handler = authentication.GetInternalAuthHandler(c.Type)
	case c.LocalAccountID.Valid:
		owner = &LocalAccount{ID: c.LocalAccountID.Int64}
		handler = authentication.GetInternalAuthHandler(c.Type)
	case c.RemoteAccountID.Valid:
		owner = &RemoteAccount{ID: c.RemoteAccountID.Int64}
		handler = authentication.GetExternalAuthMethod(c.Type)
	default:
		return nil, nil, database.NewValidationError("the authentication method is missing an owner")
	}

	if handler == nil {
		return nil, nil, database.NewValidationError("unknown auth type %q", c.Type)
	}

	if err := db.Get(owner, "id=?", owner.GetID()).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, nil, database.NewValidationError(`no %s found with ID "%v"`,
				owner.Appellation(), owner.GetID())
		}

		return nil, nil, fmt.Errorf("failed to retrieve %s credential owner: %w",
			owner.Appellation(), err)
	}

	return owner, handler, nil
}

func (c *Credential) validate(db database.ReadAccess) error {
	owner, handler, err := c.getOwner(db)
	if err != nil {
		return err
	}

	if n, err := db.Count(c).Where(owner.GetCredCond()).Where("id<>? AND name=?",
		c.ID, c.Name).Run(); err != nil {
		return fmt.Errorf("failed to check for duplicate credentials: %w", err)
	} else if n > 0 {
		return database.NewValidationError("an authentication method with the same "+
			"name %q already exist", c.Name)
	}

	if handler.CanOnlyHaveOne() {
		if n, err := db.Count(c).Where(owner.GetCredCond()).Where("type=? AND id<>?",
			c.Type, c.ID).Run(); err != nil {
			return fmt.Errorf("failed to check for duplicate credentials: %w", err)
		} else if n > 0 {
			return database.NewValidationError("this %s already has a %s authentication method",
				owner.Appellation(), c.Name)
		}
	}

	if err := handler.Validate(c.Value, c.Value2, owner.Host(), owner.IsServer()); err != nil {
		return database.NewValidationError("failed to validate authentication value: %s", err)
	}

	if ser, ok := handler.(authentication.Serializer); ok {
		var err error
		if c.Value, c.Value2, err = ser.ToDB(c.Value, c.Value2); err != nil {
			return database.NewValidationError("failed to serialize the authentication value: %s", err)
		}
	}

	return nil
}

// AfterWrite re-deserializes the authentication value (when it is relevant).
func (c *Credential) AfterWrite(database.Access) error {
	return c.AfterRead(nil)
}

// AfterRead deserializes the authentication value (when it is relevant).
func (c *Credential) AfterRead(database.ReadAccess) error {
	var handler authentication.Handler

	switch {
	case c.LocalAgentID.Valid, c.RemoteAccountID.Valid:
		handler = authentication.GetExternalAuthMethod(c.Type)
	case c.RemoteAgentID.Valid, c.LocalAccountID.Valid:
		handler = authentication.GetInternalAuthHandler(c.Type)
	default:
		return database.NewValidationError("the authentication method is missing an owner")
	}

	if handler == nil {
		return database.NewValidationError("unknown auth type %q", c.Type)
	}

	if des, ok := handler.(authentication.Deserializer); ok {
		var err error

		if c.Value, c.Value2, err = des.FromDB(c.Value, c.Value2); err != nil {
			return database.NewValidationError("failed to deserialize the authentication value: %s", err)
		}
	}

	return nil
}
