package model

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

const (
	ebicsStandardBTFEntryStatusActive   = "ACTIVE"
	ebicsStandardBTFEntryStatusDisabled = "DISABLED"
)

// EbicsStandardBTFEntry stores one standard tuple/template attached to a standard catalog.
type EbicsStandardBTFEntry struct {
	ID    int64  `xorm:"<- id AUTOINCR"`
	Owner string `xorm:"owner"`

	CatalogID         int64  `xorm:"catalog_id"`
	EntryKey          string `xorm:"entry_key"`
	OrderType         string `xorm:"order_type"`
	Direction         string `xorm:"direction"`
	ServiceName       string `xorm:"service_name"`
	ServiceOption     string `xorm:"service_option"`
	Scope             string `xorm:"scope"`
	MsgName           string `xorm:"msg_name"`
	ContainerType     string `xorm:"container_type"`
	CountryGroup      string `xorm:"country_group"`
	IsDefaultTemplate bool   `xorm:"is_default_template"`
	Status            string `xorm:"status"`
	Metadata          string `xorm:"metadata"`

	CreatedAt time.Time `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt time.Time `xorm:"updated_at UPDATED DATETIME(6) UTC"`

	MetadataMap map[string]any `xorm:"-"`
}

func (*EbicsStandardBTFEntry) TableName() string   { return TableEbicsStandardBTFEntries }
func (*EbicsStandardBTFEntry) Appellation() string { return NameEbicsStandardBTFEntry }
func (e *EbicsStandardBTFEntry) GetID() int64      { return e.ID }

func (e *EbicsStandardBTFEntry) BeforeWrite(db database.Access) error {
	e.Owner = conf.GlobalConfig.GatewayName
	e.EntryKey = strings.TrimSpace(e.EntryKey)
	e.OrderType = NormalizeEbicsPayloadOrderType(e.OrderType)
	e.Direction = strings.ToUpper(strings.TrimSpace(e.Direction))
	e.ServiceName = strings.TrimSpace(e.ServiceName)
	e.ServiceOption = strings.TrimSpace(e.ServiceOption)
	e.Scope = strings.ToUpper(strings.TrimSpace(e.Scope))
	e.MsgName = strings.TrimSpace(e.MsgName)
	e.ContainerType = strings.TrimSpace(e.ContainerType)
	e.CountryGroup = strings.ToUpper(strings.TrimSpace(e.CountryGroup))
	e.Status = strings.ToUpper(strings.TrimSpace(e.Status))
	e.Metadata = strings.TrimSpace(e.Metadata)

	if e.CatalogID == 0 {
		return database.NewValidationError("the EBICS standard BTF catalog reference is missing")
	}
	if e.EntryKey == "" {
		return database.NewValidationError("the EBICS standard BTF entry key cannot be empty")
	}
	if err := validateEbicsPayloadOrderType(e.OrderType); err != nil {
		return err
	}
	if err := validateEbicsPayloadDirection(e.Direction); err != nil {
		return err
	}
	if e.ServiceName == "" {
		return database.NewValidationError("the EBICS standard BTF service name is missing")
	}
	if err := validateEbicsStandardBTFCatalogScope(e.Scope); err != nil {
		return err
	}
	if e.CountryGroup != "" {
		if err := validateEbicsStandardBTFCatalogScope(e.CountryGroup); err != nil {
			return err
		}
	}
	if err := validateEbicsStandardBTFEntryStatus(e.Status); err != nil {
		return err
	}

	if e.MetadataMap != nil {
		serialized, err := json.Marshal(e.MetadataMap)
		if err != nil {
			return database.NewValidationErrorf("invalid EBICS standard BTF metadata: %w", err)
		}
		e.Metadata = string(serialized)
	} else if e.Metadata == "" {
		e.Metadata = emptyJSONObject
	}

	var catalog EbicsStandardBTFCatalog
	if err := db.Get(&catalog, "id=?", e.CatalogID).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf("the EBICS standard BTF catalog %d does not exist", e.CatalogID)
		}

		return fmt.Errorf("failed to retrieve EBICS standard BTF catalog %d: %w", e.CatalogID, err)
	}

	if n, err := db.Count(e).Where(
		"id<>? AND owner=? AND catalog_id=? AND entry_key=?",
		e.ID,
		e.Owner,
		e.CatalogID,
		e.EntryKey,
	).Run(); err != nil {
		return fmt.Errorf("failed to check duplicate EBICS standard BTF entries: %w", err)
	} else if n != 0 {
		return database.NewValidationErrorf(
			"an EBICS standard BTF entry %q already exists in catalog %d",
			e.EntryKey,
			e.CatalogID,
		)
	}

	return e.AfterRead(db)
}

func (e *EbicsStandardBTFEntry) AfterRead(database.ReadAccess) error {
	raw := strings.TrimSpace(e.Metadata)
	if raw == "" {
		e.MetadataMap = map[string]any{}
		return nil
	}

	if err := json.Unmarshal([]byte(raw), &e.MetadataMap); err != nil {
		return database.NewValidationErrorf("invalid serialized EBICS standard BTF metadata: %w", err)
	}
	if e.MetadataMap == nil {
		e.MetadataMap = map[string]any{}
	}

	return nil
}

func (e *EbicsStandardBTFEntry) AfterInsert(db database.Access) error {
	return e.AfterRead(db)
}

func (e *EbicsStandardBTFEntry) AfterUpdate(db database.Access) error {
	return e.AfterRead(db)
}

func validateEbicsStandardBTFEntryStatus(status string) error {
	switch status {
	case ebicsStandardBTFEntryStatusActive, ebicsStandardBTFEntryStatusDisabled:
		return nil
	case "":
		return database.NewValidationError("the EBICS standard BTF entry status is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS standard BTF entry status", status)
	}
}
