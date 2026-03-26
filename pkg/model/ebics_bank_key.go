package model

import (
	"fmt"
	"strings"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

const (
	ebicsBankKeyTypeAuth      = "AUTH"
	ebicsBankKeyTypeEncrypt   = "ENCRYPT"
	ebicsBankKeyTypeSignature = "SIGNATURE"

	ebicsBankKeyStateImported  = "imported"
	ebicsBankKeyStateValidated = "validated"
	ebicsBankKeyStateRetired   = "retired"
)

type EbicsBankKey struct {
	ID    int64  `xorm:"<- id AUTOINCR"`
	Owner string `xorm:"owner"`

	EbicsHostID int64 `xorm:"ebics_host_id"`

	KeyType       string `xorm:"key_type"`
	Version       string `xorm:"version"`
	PublicKey     string `xorm:"public_key"`
	PublicKeyHash string `xorm:"public_key_hash"`
	State         string `xorm:"state"`

	ValidFrom time.Time `xorm:"valid_from DATETIME(6) UTC"`
	ValidTo   time.Time `xorm:"valid_to DATETIME(6) UTC"`
	CreatedAt time.Time `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt time.Time `xorm:"updated_at UPDATED DATETIME(6) UTC"`
}

func (*EbicsBankKey) TableName() string   { return TableEbicsBankKeys }
func (*EbicsBankKey) Appellation() string { return NameEbicsBankKey }
func (k *EbicsBankKey) GetID() int64      { return k.ID }

func (k *EbicsBankKey) BeforeWrite(db database.Access) error {
	k.Owner = conf.GlobalConfig.GatewayName
	k.KeyType = strings.ToUpper(strings.TrimSpace(k.KeyType))
	k.Version = strings.TrimSpace(k.Version)
	k.PublicKey = strings.TrimSpace(k.PublicKey)
	k.PublicKeyHash = strings.TrimSpace(k.PublicKeyHash)
	k.State = strings.ToLower(strings.TrimSpace(k.State))

	if k.EbicsHostID == 0 {
		return database.NewValidationError("the EBICS host reference is missing")
	}

	if err := validateEbicsBankKeyType(k.KeyType); err != nil {
		return err
	}

	if err := validateEbicsBankKeyState(k.State); err != nil {
		return err
	}

	if k.PublicKey == "" {
		return database.NewValidationError("the EBICS bank public key is missing")
	}

	var host EbicsHost
	if err := db.Get(&host, "id=?", k.EbicsHostID).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf("the EBICS host %d does not exist", k.EbicsHostID)
		}

		return fmt.Errorf("failed to retrieve EBICS host: %w", err)
	}

	if n, err := db.Count(k).Where(
		"id<>? AND owner=? AND ebics_host_id=? AND key_type=? AND version=?",
		k.ID, k.Owner, k.EbicsHostID, k.KeyType, k.Version,
	).Run(); err != nil {
		return fmt.Errorf("failed to check duplicate EBICS bank keys: %w", err)
	} else if n != 0 {
		return database.NewValidationErrorf(
			"an EBICS bank key already exists for type %q and version %q", k.KeyType, k.Version)
	}

	return nil
}

func validateEbicsBankKeyType(keyType string) error {
	switch keyType {
	case ebicsBankKeyTypeAuth, ebicsBankKeyTypeEncrypt, ebicsBankKeyTypeSignature:
		return nil
	case "":
		return database.NewValidationError("the EBICS bank key type is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS bank key type", keyType)
	}
}

func validateEbicsBankKeyState(state string) error {
	switch state {
	case ebicsBankKeyStateImported, ebicsBankKeyStateValidated, ebicsBankKeyStateRetired:
		return nil
	case "":
		return database.NewValidationError("the EBICS bank key state is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS bank key state", state)
	}
}
