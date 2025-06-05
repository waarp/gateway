package model

import (
	"fmt"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

type Client struct {
	ID    int64  `xorm:"<- id AUTOINCR"` // The client's database ID.
	Owner string `xorm:"owner"`          // The client's owner (the gateway to which it belongs)

	Name         string        `xorm:"name"`          // The client's name.
	Protocol     string        `xorm:"protocol"`      // The client's protocol.
	LocalAddress types.Address `xorm:"local_address"` // The client's local address (optional).

	// The client's protocol configuration as a map.
	ProtoConfig map[string]any `xorm:"proto_config"`

	Disabled bool // Should the client be launched at startup.
}

func (c *Client) GetID() int64      { return c.ID }
func (*Client) TableName() string   { return TableClients }
func (*Client) Appellation() string { return "client" }

func (c *Client) BeforeWrite(db database.Access) error {
	c.Owner = conf.GlobalConfig.GatewayName

	if c.Name == "" {
		c.Name = c.Protocol
	}

	if strings.TrimSpace(c.Name) == "" {
		return database.NewValidationError("the client's name cannot be empty")
	}

	if strings.TrimSpace(c.Protocol) == "" {
		return database.NewValidationError("the client's protocol is missing")
	} else if !ConfigChecker.IsValidProtocol(c.Protocol) {
		return database.NewValidationErrorf("%q is not a protocol", c.Protocol)
	}

	if c.LocalAddress.IsSet() {
		if err := c.LocalAddress.Validate(); err != nil {
			return database.NewValidationErrorf("address validation failed: %w", err)
		}
	}

	if c.ProtoConfig == nil {
		c.ProtoConfig = map[string]any{}
	}

	if err := ConfigChecker.CheckClientConfig(c.Protocol, c.ProtoConfig); err != nil {
		return database.WrapAsValidationError(err)
	}

	if n, err := db.Count(c).Where("id<>? AND owner=? AND name=?", c.ID, c.Owner,
		c.Name).Run(); err != nil {
		return fmt.Errorf("failed to check for duplicate clients: %w", err)
	} else if n != 0 {
		return database.NewValidationErrorf("a client named %q already exist", c.Name)
	}

	return nil
}
