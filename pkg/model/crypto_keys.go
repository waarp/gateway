package model

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

type CryptoKey struct {
	ID    int64               `xorm:"<- id AUTOINCR"`
	Owner string              `xorm:"owner"`
	Name  string              `xorm:"name"`
	Type  string              `xorm:"type"`
	Key   database.SecretText `xorm:"value"`
}

func (*CryptoKey) TableName() string   { return TableCryptoKeys }
func (*CryptoKey) Appellation() string { return NameCryptoKey }
func (k *CryptoKey) GetID() int64      { return k.ID }

func (k *CryptoKey) BeforeWrite(db database.Access) error {
	k.Owner = conf.GlobalConfig.GatewayName

	if k.Name == "" {
		return database.NewValidationError("the cryptographic key's name is missing")
	}

	if k.Type == "" {
		return database.NewValidationError("the cryptographic key's type is missing")
	}

	if k.Key == "" {
		return database.NewValidationError("the cryptographic key value is missing")
	}

	if n, err := db.Count(k).Where("name=? AND id<>?", k.Name, k.ID).Run(); err != nil {
		return fmt.Errorf("failed to check existing cryptographic keys: %w", err)
	} else if n != 0 {
		return database.NewValidationError("a cryptographic key named %q already exists", k.Name)
	}

	return k.checkKey()
}
