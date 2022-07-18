package model

import (
	"database/sql"
	"encoding/json"
	"net"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/names"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
)

type Client struct {
	ID           uint64          `xorm:"BIGINT PK AUTOINCR <- 'id'"`
	Owner        string          `xorm:"VARCHAR(100) UNIQUE(client) NOTNULL 'owner'"`
	Name         string          `xorm:"VARCHAR(100) UNIQUE(client) NOTNULL 'name'"`
	Protocol     string          `xorm:"VARCHAR(20) NOTNULL 'protocol'"`
	LocalAddress sql.NullString  `xorm:"VARCHAR(255) 'local_address'"`
	ProtoConfig  json.RawMessage `xorm:"TEXT NOTNULL DEFAULT('{}') 'proto_config'"`
}

func (c *Client) GetID() uint64     { return c.ID }
func (*Client) TableName() string   { return TableClients }
func (*Client) Appellation() string { return "client" }

func (c *Client) BeforeWrite(db database.ReadAccess) database.Error {
	c.Owner = conf.GlobalConfig.GatewayName

	if strings.TrimSpace(c.Name) == "" {
		return database.NewValidationError("the client's name cannot be empty")
	} else if names.IsReservedServiceName(c.Name) {
		return database.NewValidationError("%q is a reserved service name", c.Name)
	}

	if strings.TrimSpace(c.Protocol) == "" {
		return database.NewValidationError("the client's protocol cannot be empty")
	} else {
		if _, ok := config.ProtoConfigs[c.Protocol]; !ok {
			return database.NewValidationError("%q is not a protocol", c.Protocol)
		}
	}

	if c.LocalAddress.Valid {
		if _, err := net.LookupHost(c.LocalAddress.String); err != nil {
			return database.NewValidationError("%q is not a valid client address: %v",
				c.LocalAddress.String, err)
		}
	}

	if protoConf, parseErr := config.ParseClientConfig(c.Protocol, c.ProtoConfig); parseErr != nil {
		return database.NewValidationError("%v", parseErr)
	} else if validErr := protoConf.ValidClient(); validErr != nil {
		return database.NewValidationError("client proto config validation failed: %v", validErr)
	}

	if n, err := db.Count(c).Where("id<>? AND owner=? AND name=?", c.ID, c.Owner,
		c.Name).Run(); err != nil {
		return err
	} else if n != 0 {
		return database.NewValidationError("a client named %q already exist", c.Name)
	}

	return nil
}
