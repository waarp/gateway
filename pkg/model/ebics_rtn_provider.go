package model

import (
	"fmt"
	"strings"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

const (
	ebicsRTNTransportWSS = "WSS"

	ebicsRTNAutoPullPolicyManual       = "MANUAL"
	ebicsRTNAutoPullPolicyAuto         = "AUTO"
	ebicsRTNAutoPullPolicyAutoFiltered = "AUTO_FILTERED"
)

// EbicsRTNProvider stores the administrable configuration of an RTN provider.
type EbicsRTNProvider struct {
	ID                int64     `xorm:"<- id AUTOINCR"`
	Owner             string    `xorm:"owner"`
	Name              string    `xorm:"name"`
	Transport         string    `xorm:"transport"`
	Enabled           bool      `xorm:"enabled"`
	EbicsSubscriberID int64     `xorm:"ebics_subscriber_id"`
	Configuration     string    `xorm:"configuration"`
	AutoPullPolicy    string    `xorm:"auto_pull_policy"`
	LastConnectionAt  time.Time `xorm:"last_connection_at DATETIME(6) UTC"`
	LastError         string    `xorm:"last_error"`
	CreatedAt         time.Time `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt         time.Time `xorm:"updated_at UPDATED DATETIME(6) UTC"`

	ConfigurationMap map[string]any `xorm:"-"`
}

// TableName returns the persistent table name for RTN providers.
func (*EbicsRTNProvider) TableName() string { return TableEbicsRTNProviders }

// Appellation returns the display name used in validation messages.
func (*EbicsRTNProvider) Appellation() string { return NameEbicsRTNProvider }

// GetID returns the database identifier of the provider.
func (p *EbicsRTNProvider) GetID() int64 { return p.ID }

// BeforeWrite normalizes and validates an RTN provider before persistence.
func (p *EbicsRTNProvider) BeforeWrite(db database.Access) error {
	p.Owner = conf.GlobalConfig.GatewayName
	p.Name = strings.TrimSpace(p.Name)
	p.Transport = strings.ToUpper(strings.TrimSpace(p.Transport))
	p.Configuration = strings.TrimSpace(p.Configuration)
	p.AutoPullPolicy = strings.ToUpper(strings.TrimSpace(p.AutoPullPolicy))
	p.LastError = strings.TrimSpace(p.LastError)

	if p.Name == "" {
		return database.NewValidationError("the RTN provider name is missing")
	}

	if p.EbicsSubscriberID == 0 {
		return database.NewValidationError("the RTN provider subscriber reference is missing")
	}

	if err := validateEbicsRTNTransport(p.Transport); err != nil {
		return err
	}

	if err := validateEbicsRTNAutoPullPolicy(p.AutoPullPolicy); err != nil {
		return err
	}

	if err := p.hydrateConfiguration(); err != nil {
		return err
	}

	if endpoint, ok := readRTNProviderConfigString(p.ConfigurationMap, "endpoint"); !ok || endpoint == "" {
		return database.NewValidationError("the RTN provider endpoint is missing")
	}
	if p.AutoPullPolicy != ebicsRTNAutoPullPolicyManual {
		clientID, ok := readRTNProviderConfigInt64(p.ConfigurationMap, "clientID")
		if !ok || clientID == 0 {
			return database.NewValidationError("the RTN provider client ID is missing")
		}
	}

	if err := validateEbicsRTNProviderRefs(db, p); err != nil {
		return err
	}

	return validateEbicsRTNProviderUniqueness(db, p)
}

func (p *EbicsRTNProvider) hydrateConfiguration() error {
	if p.ConfigurationMap != nil {
		serialized, err := serializeStringMap(p.ConfigurationMap)
		if err != nil {
			return fmt.Errorf("failed to serialize RTN provider configuration: %w", err)
		}

		p.Configuration = serialized
	} else if p.Configuration == "" {
		p.Configuration = emptyJSONObject
	}

	config, err := deserializeStringMap(p.Configuration)
	if err != nil {
		return fmt.Errorf("failed to deserialize RTN provider configuration: %w", err)
	}

	p.ConfigurationMap = config

	return nil
}

// AfterRead hydrates the transient configuration map after a database read.
func (p *EbicsRTNProvider) AfterRead(database.ReadAccess) error {
	config, err := deserializeStringMap(p.Configuration)
	if err != nil {
		return fmt.Errorf("failed to deserialize RTN provider configuration after read: %w", err)
	}

	p.ConfigurationMap = config

	return nil
}

// AfterInsert refreshes transient state after insertion.
func (p *EbicsRTNProvider) AfterInsert(db database.Access) error { return p.AfterRead(db) }

// AfterUpdate refreshes transient state after update.
func (p *EbicsRTNProvider) AfterUpdate(db database.Access) error { return p.AfterRead(db) }

func validateEbicsRTNTransport(value string) error {
	switch value {
	case ebicsRTNTransportWSS:
		return nil
	case "":
		return database.NewValidationError("the RTN provider transport is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported RTN transport", value)
	}
}

func validateEbicsRTNAutoPullPolicy(value string) error {
	switch value {
	case ebicsRTNAutoPullPolicyManual, ebicsRTNAutoPullPolicyAuto, ebicsRTNAutoPullPolicyAutoFiltered:
		return nil
	case "":
		return database.NewValidationError("the RTN provider auto-pull policy is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported RTN auto-pull policy", value)
	}
}

func validateEbicsRTNProviderRefs(db database.Access, provider *EbicsRTNProvider) error {
	var subscriber EbicsSubscriber
	if err := db.Get(&subscriber, "id=?", provider.EbicsSubscriberID).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf(
				"the EBICS subscriber %d does not exist", provider.EbicsSubscriberID)
		}

		return fmt.Errorf("failed to retrieve EBICS subscriber for RTN provider: %w", err)
	}

	if provider.AutoPullPolicy != ebicsRTNAutoPullPolicyManual {
		clientID, _ := readRTNProviderConfigInt64(provider.ConfigurationMap, "clientID")
		var client Client
		if err := db.Get(&client, "id=?", clientID).Run(); err != nil {
			if database.IsNotFound(err) {
				return database.NewValidationErrorf("the EBICS client %d does not exist", clientID)
			}

			return fmt.Errorf("failed to retrieve EBICS client for RTN provider: %w", err)
		}
		if client.Protocol != "ebics" {
			return database.NewValidationErrorf("the RTN provider client %d does not use the EBICS protocol", clientID)
		}
		if client.Disabled {
			return database.NewValidationErrorf("the RTN provider client %d is disabled", clientID)
		}
	}

	return nil
}

func validateEbicsRTNProviderUniqueness(db database.Access, provider *EbicsRTNProvider) error {
	count, err := db.Count(provider).Where(
		"id<>? AND owner=? AND name=?",
		provider.ID, provider.Owner, provider.Name,
	).Run()
	if err != nil {
		return fmt.Errorf("failed to check duplicate RTN providers: %w", err)
	}

	if count != 0 {
		return database.NewValidationErrorf("an RTN provider named %q already exists", provider.Name)
	}

	return nil
}

func readRTNProviderConfigString(config map[string]any, key string) (string, bool) {
	if config == nil {
		return "", false
	}

	value, ok := config[key]
	if !ok {
		return "", false
	}

	raw, ok := value.(string)
	if !ok {
		return "", false
	}

	return strings.TrimSpace(raw), true
}

func readRTNProviderConfigInt64(config map[string]any, key string) (int64, bool) {
	if config == nil {
		return 0, false
	}

	value, ok := config[key]
	if !ok {
		return 0, false
	}

	switch raw := value.(type) {
	case int64:
		return raw, true
	case int:
		return int64(raw), true
	case float64:
		return int64(raw), true
	default:
		return 0, false
	}
}
