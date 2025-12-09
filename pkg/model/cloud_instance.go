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

	oldName string
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
		return database.NewValidationErrorf("invalid cloud instance configuration: %v", err)
	}

	if n, err := db.Count(c).Where("id<>? AND owner=? AND name=?", c.ID, c.Owner,
		c.Name).Run(); err != nil {
		return fmt.Errorf("failed to check existing cloud instances: %w", err)
	} else if n > 0 {
		return database.NewValidationErrorf("a cloud instance named %q already exist", c.Name)
	}

	return nil
}

func (c *CloudInstance) AfterInsert(database.Access) error {
	fileSys, err := fs.NewFS(c.Name, c.Type, c.Key, c.Secret.String(), c.Options)
	if err != nil {
		return database.NewValidationErrorf("invalid cloud instance configuration: %v", err)
	}

	fs.FileSystems.Store(c.Name, fileSys)

	return nil
}

func (c *CloudInstance) AfterUpdate(db database.Access) error {
	if err := c.AfterInsert(db); err != nil {
		return err
	}

	if c.oldName != c.Name {
		fs.FileSystems.Delete(c.oldName)
	}

	return nil
}

func (c *CloudInstance) AfterDelete(database.Access) error {
	fs.FileSystems.Delete(c.Name)

	return nil
}

func (c *CloudInstance) AfterRead(database.ReadAccess) error {
	c.oldName = c.Name

	return nil
}
