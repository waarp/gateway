package model

import (
	"fmt"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

//nolint:lll //descriptions exceed the max length but are more readable this way
type Client struct {
	Identifier
	Owner    string `gorm:"column:owner"`    // The client's owner (the gateway to which it belongs)
	Name     string `gorm:"column:name"`     // The client's name.
	Protocol string `gorm:"column:protocol"` // The client's protocol.

	LocalAddress         types.Address `gorm:"column:local_address"`          // The client's local address (optional).
	NbOfAttempts         int8          `gorm:"column:nb_of_attempts"`         // The number of times the client will automatically re-attempt a transfer.
	FirstRetryDelay      int32         `gorm:"column:first_retry_delay"`      // The delay (in seconds) between the original attempt and the first re-attempt.
	RetryIncrementFactor float32       `gorm:"column:retry_increment_factor"` // The factor by which the delay will be multiplied at each re-attempt.

	// The client's protocol configuration as a map.
	ProtoConfig Map[any] `gorm:"column:proto_config;serializer:json"`

	Disabled bool `gorm:"column:disabled"` // Should the client be launched at startup.
}

func (*Client) TableName() string   { return TableClients }
func (*Client) Appellation() string { return "client" }

func (c *Client) BeforeWrite(db database.Access) error {
	if c.Name == "" {
		c.Name = c.Protocol
	}

	if c.FirstRetryDelay != 0 && c.RetryIncrementFactor == 0 {
		c.RetryIncrementFactor = 1.0
	}

	if strings.TrimSpace(c.Name) == "" {
		return database.NewValidationError("the client's name cannot be empty")
	}

	if strings.TrimSpace(c.Protocol) == "" {
		return database.NewValidationError("the client's protocol is missing")
	}

	if c.LocalAddress.IsSet() {
		if err := c.LocalAddress.Validate(); err != nil {
			return database.NewValidationErrorf("address validation failed: %w", err)
		}
	}

	if c.ProtoConfig == nil {
		c.ProtoConfig = Map[any]{}
	}

	if err := CheckClientConfig(c.Protocol, c.ProtoConfig); err != nil {
		return database.WrapAsValidationError(err)
	}

	if n, err := db.Count(c).Where("id<>? AND name=?", c.ID, c.Name).Run(); err != nil {
		return fmt.Errorf("failed to check for duplicate clients: %w", err)
	} else if n != 0 {
		return database.NewValidationErrorf("a client named %q already exist", c.Name)
	}

	return nil
}
