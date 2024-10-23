package model

import (
	"fmt"

	pgp "github.com/ProtonMail/gopenpgp/v3/crypto"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

type PGPKey struct {
	ID         int64               `xorm:"<- id AUTOINCR"`
	Name       string              `xorm:"name"`
	PrivateKey database.SecretText `xorm:"private_key"`
	PublicKey  string              `xorm:"public_key"`
}

func (*PGPKey) TableName() string   { return TablePGPKeys }
func (*PGPKey) Appellation() string { return NamePGPKey }
func (k *PGPKey) GetID() int64      { return k.ID }

func (k *PGPKey) BeforeWrite(database.Access) error {
	if k.Name == "" {
		return database.NewValidationError("the PGP key's name is missing")
	}

	if k.PrivateKey == "" && k.PublicKey == "" {
		return database.NewValidationError("the PGP private and/or public key is missing")
	}

	if k.PrivateKey != "" {
		privKey, privErr := pgp.NewKeyFromArmored(string(k.PrivateKey))
		if privErr != nil {
			return fmt.Errorf("failed to parse PGP private key: %w", privErr)
		}

		if !privKey.IsPrivate() {
			return database.NewValidationError("the given PGP private key is not a private key")
		}

		if k.PublicKey == "" {
			if pubKey, pubErr := privKey.ToPublic(); pubErr != nil {
				return fmt.Errorf("failed to extract PGP public key from private key: %w", pubErr)
			} else if k.PublicKey, pubErr = pubKey.Armor(); pubErr != nil {
				return fmt.Errorf("failed to serialize extracted PGP public key: %w", pubErr)
			}
		}
	}

	pubKey, err := pgp.NewKeyFromArmored(k.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to parse PGP public key: %w", err)
	}

	if pubKey.IsPrivate() {
		return database.NewValidationError("the given PGP public key is not a public key")
	}

	return nil
}
