package model

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

type CryptoKey struct {
	Identifier
	Owner string `gorm:"column:owner"`
	Name  string `gorm:"column:name"`
	Type  string `gorm:"column:type"`
	Key   string `gorm:"column:value;serializer:secret"`
}

func (*CryptoKey) TableName() string   { return TableCryptoKeys }
func (*CryptoKey) Appellation() string { return NameCryptoKey }

func (k *CryptoKey) BeforeWrite(db database.Access) error {
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
		return database.NewValidationErrorf("a cryptographic key named %q already exists", k.Name)
	}

	return k.checkKey()
}
