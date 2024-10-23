package model

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
)

type CloudInstance struct {
	ID      int64               `xorm:"<- id AUTOINCR"`
	Owner   string              `xorm:"owner"`
	Name    string              `xorm:"name"`
	Type    string              `xorm:"type"`
	Key     string              `xorm:"api_key"`
	Secret  database.SecretText `xorm:"secret"`
	Options map[string]string   `xorm:"options"`
}

func (c *CloudInstance) TableName() string   { return TableCloudInstances }
func (c *CloudInstance) Appellation() string { return "cloud instance" }
func (c *CloudInstance) GetID() int64        { return c.ID }

func (c *CloudInstance) BeforeWrite(db database.Access) error {
	c.Owner = conf.GlobalConfig.GatewayName

	if c.Name == "" {
		return database.NewValidationError("the cloud instance's name cannot be empty")
	}

	if c.Name == "file" {
		return database.NewValidationError(`the name "file" is reserved for the local filesystem`)
	}

	if err := fs.ValidateConfig(c.Name, c.Type, c.Key, c.Secret.String(), c.Options); err != nil {
		return database.NewValidationError("invalid cloud instance configuration: %v", err)
	}

	if n, err := db.Count(c).Where("id<>? AND owner=? AND name=?", c.ID, c.Owner,
		c.Name).Run(); err != nil {
		return fmt.Errorf("failed to check existing cloud instances: %w", err)
	} else if n > 0 {
		return database.NewValidationError("a cloud instance named %q already exist", c.Name)
	}

	return nil
}
