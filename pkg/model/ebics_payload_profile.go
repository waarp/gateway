package model

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

const (
	ebicsPayloadOrderBTU = "BTU"
	ebicsPayloadOrderBTD = "BTD"
	ebicsPayloadOrderFUL = "FUL"
	ebicsPayloadOrderFDL = "FDL"

	ebicsPayloadDirectionUpload        = "UPLOAD"
	ebicsPayloadDirectionDownload      = "DOWNLOAD"
	ebicsPayloadDirectionBidirectional = "BIDIRECTIONAL"

	emptyJSONArray  = "[]"
	emptyJSONObject = "{}"
)

type EbicsPayloadProfile struct {
	ID    int64  `xorm:"<- id AUTOINCR"`
	Owner string `xorm:"owner"`

	Name                   string        `xorm:"name"`
	Label                  string        `xorm:"label"`
	Description            string        `xorm:"description"`
	OrderType              string        `xorm:"order_type"`
	Direction              string        `xorm:"direction"`
	ServiceName            string        `xorm:"service_name"`
	ServiceOption          string        `xorm:"service_option"`
	Scope                  string        `xorm:"scope"`
	MsgName                string        `xorm:"msg_name"`
	ContainerType          string        `xorm:"container_type"`
	DefaultRuleID          sql.NullInt64 `xorm:"default_rule_id"`
	DefaultTargetDirectory string        `xorm:"default_target_directory"`
	RequiresDeclaredAmount bool          `xorm:"requires_declared_amount"`
	DefaultCurrency        string        `xorm:"default_currency"`
	AllowedExtensions      string        `xorm:"allowed_extensions"`
	FilenamePattern        string        `xorm:"filename_pattern"`
	StrictContractCheck    bool          `xorm:"strict_contract_check"`
	IsEnabled              bool          `xorm:"is_enabled"`
	Metadata               string        `xorm:"metadata"`
	CreatedAt              time.Time     `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt              time.Time     `xorm:"updated_at UPDATED DATETIME(6) UTC"`

	DefaultRuleName       string         `xorm:"-"`
	AllowedExtensionsList []string       `xorm:"-"`
	MetadataMap           map[string]any `xorm:"-"`
}

func (*EbicsPayloadProfile) TableName() string   { return TableEbicsPayloadProfiles }
func (*EbicsPayloadProfile) Appellation() string { return NameEbicsPayloadProfile }
func (p *EbicsPayloadProfile) GetID() int64      { return p.ID }

func (p *EbicsPayloadProfile) BeforeWrite(db database.Access) error {
	p.Owner = conf.GlobalConfig.GatewayName
	p.Name = strings.TrimSpace(p.Name)
	p.Label = strings.TrimSpace(p.Label)
	p.Description = strings.TrimSpace(p.Description)
	p.OrderType = strings.ToUpper(strings.TrimSpace(p.OrderType))
	p.Direction = strings.ToUpper(strings.TrimSpace(p.Direction))
	p.ServiceName = strings.TrimSpace(p.ServiceName)
	p.ServiceOption = strings.TrimSpace(p.ServiceOption)
	p.Scope = strings.TrimSpace(p.Scope)
	p.MsgName = strings.TrimSpace(p.MsgName)
	p.ContainerType = strings.TrimSpace(p.ContainerType)
	p.DefaultTargetDirectory = strings.TrimSpace(p.DefaultTargetDirectory)
	p.DefaultCurrency = strings.ToUpper(strings.TrimSpace(p.DefaultCurrency))
	p.FilenamePattern = strings.TrimSpace(p.FilenamePattern)
	p.AllowedExtensions = strings.TrimSpace(p.AllowedExtensions)
	p.Metadata = strings.TrimSpace(p.Metadata)

	if p.Name == "" {
		return database.NewValidationError("the EBICS payload profile name cannot be empty")
	}

	if err := validateEbicsPayloadOrderType(p.OrderType); err != nil {
		return err
	}

	if err := validateEbicsPayloadDirection(p.Direction); err != nil {
		return err
	}

	if err := validateEbicsPayloadProfileCoherence(p); err != nil {
		return err
	}

	if p.AllowedExtensionsList != nil {
		serialized, err := serializeStringSlice(p.AllowedExtensionsList)
		if err != nil {
			return err
		}

		p.AllowedExtensions = serialized
	} else if p.AllowedExtensions == "" {
		p.AllowedExtensions = emptyJSONArray
	}

	if p.MetadataMap != nil {
		serialized, err := serializeStringMap(p.MetadataMap)
		if err != nil {
			return err
		}

		p.Metadata = serialized
	} else if p.Metadata == "" {
		p.Metadata = emptyJSONObject
	}

	exts, err := deserializeStringSlice(p.AllowedExtensions)
	if err != nil {
		return err
	}

	p.AllowedExtensionsList = exts

	meta, err := deserializeStringMap(p.Metadata)
	if err != nil {
		return err
	}

	p.MetadataMap = meta

	if p.DefaultRuleID.Valid {
		var rule Rule
		if err = db.Get(&rule, "id=?", p.DefaultRuleID.Int64).Run(); err != nil {
			if database.IsNotFound(err) {
				return database.NewValidationErrorf(
					"the default Gateway rule %d does not exist", p.DefaultRuleID.Int64)
			}

			return fmt.Errorf("failed to retrieve default Gateway rule: %w", err)
		}
	}

	if n, errCount := db.Count(p).Where(
		"id<>? AND owner=? AND name=?", p.ID, p.Owner, p.Name,
	).Run(); errCount != nil {
		return fmt.Errorf("failed to check duplicate EBICS payload profiles: %w", errCount)
	} else if n != 0 {
		return database.NewValidationErrorf("an EBICS payload profile named %q already exists", p.Name)
	}

	return nil
}

func (p *EbicsPayloadProfile) AfterRead(database.ReadAccess) error {
	var err error

	p.AllowedExtensionsList, err = deserializeStringSlice(p.AllowedExtensions)
	if err != nil {
		return err
	}

	p.MetadataMap, err = deserializeStringMap(p.Metadata)
	if err != nil {
		return err
	}

	return nil
}

func (p *EbicsPayloadProfile) AfterInsert(db database.Access) error {
	return p.AfterRead(db)
}

func (p *EbicsPayloadProfile) AfterUpdate(db database.Access) error {
	return p.AfterRead(db)
}

func validateEbicsPayloadOrderType(orderType string) error {
	switch orderType {
	case ebicsPayloadOrderBTU, ebicsPayloadOrderBTD, ebicsPayloadOrderFUL, ebicsPayloadOrderFDL:
		return nil
	case "":
		return database.NewValidationError("the EBICS payload order type is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS payload order type", orderType)
	}
}

func validateEbicsPayloadDirection(direction string) error {
	switch direction {
	case ebicsPayloadDirectionUpload, ebicsPayloadDirectionDownload, ebicsPayloadDirectionBidirectional:
		return nil
	case "":
		return database.NewValidationError("the EBICS payload direction is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS payload direction", direction)
	}
}

func validateEbicsPayloadProfileCoherence(p *EbicsPayloadProfile) error {
	switch p.OrderType {
	case ebicsPayloadOrderBTU, ebicsPayloadOrderFUL:
		if p.Direction == ebicsPayloadDirectionDownload {
			return database.NewValidationError(
				"an upload EBICS payload order cannot use the DOWNLOAD direction")
		}
	case ebicsPayloadOrderBTD, ebicsPayloadOrderFDL:
		if p.Direction == ebicsPayloadDirectionUpload {
			return database.NewValidationError(
				"a download EBICS payload order cannot use the UPLOAD direction")
		}
	}

	if p.RequiresDeclaredAmount && p.DefaultCurrency == "" {
		return database.NewValidationError(
			"an EBICS payload profile requiring declared amounts also requires a default currency")
	}

	return nil
}

func serializeStringSlice(values []string) (string, error) {
	serialized, err := json.Marshal(values)
	if err != nil {
		return "", database.NewValidationErrorf(
			"invalid EBICS payload profile allowed extensions: %w", err)
	}

	return string(serialized), nil
}

func deserializeStringSlice(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}, nil
	}

	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, database.NewValidationErrorf(
			"invalid serialized EBICS payload profile allowed extensions: %w", err)
	}

	return values, nil
}

func serializeStringMap(raw map[string]any) (string, error) {
	serialized, err := json.Marshal(raw)
	if err != nil {
		return "", database.NewValidationErrorf(
			"invalid EBICS payload profile metadata: %w", err)
	}

	return string(serialized), nil
}

func deserializeStringMap(raw string) (map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}, nil
	}

	var values map[string]any
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, database.NewValidationErrorf(
			"invalid serialized EBICS payload profile metadata: %w", err)
	}

	if values == nil {
		values = map[string]any{}
	}

	return values, nil
}
