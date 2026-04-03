package model

import (
	"fmt"
	"strings"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

const (
	ebicsRTNOutboundTransportWSS = "WSS"
)

type EbicsRTNOutboundProvider struct {
	ID                int64     `xorm:"<- id AUTOINCR"`
	Owner             string    `xorm:"owner"`
	Name              string    `xorm:"name"`
	Transport         string    `xorm:"transport"`
	Enabled           bool      `xorm:"enabled"`
	EbicsSubscriberID int64     `xorm:"ebics_subscriber_id"`
	Configuration     string    `xorm:"configuration"`
	LastConnectionAt  time.Time `xorm:"last_connection_at DATETIME(6) UTC"`
	LastError         string    `xorm:"last_error"`
	CreatedAt         time.Time `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt         time.Time `xorm:"updated_at UPDATED DATETIME(6) UTC"`

	ConfigurationMap map[string]any `xorm:"-"`
}

func (*EbicsRTNOutboundProvider) TableName() string   { return TableEbicsRTNOutboundProviders }
func (*EbicsRTNOutboundProvider) Appellation() string { return NameEbicsRTNOutboundProvider }
func (p *EbicsRTNOutboundProvider) GetID() int64      { return p.ID }

func (p *EbicsRTNOutboundProvider) BeforeWrite(db database.Access) error {
	p.Owner = conf.GlobalConfig.GatewayName
	p.Name = strings.TrimSpace(p.Name)
	p.Transport = strings.ToUpper(strings.TrimSpace(p.Transport))
	p.Configuration = strings.TrimSpace(p.Configuration)
	p.LastError = strings.TrimSpace(p.LastError)

	if p.Name == "" {
		return database.NewValidationError("the outbound RTN provider name is missing")
	}
	if p.EbicsSubscriberID == 0 {
		return database.NewValidationError("the outbound RTN provider subscriber reference is missing")
	}
	if err := validateEbicsRTNOutboundTransport(p.Transport); err != nil {
		return err
	}
	if err := p.hydrateConfiguration(); err != nil {
		return err
	}
	if endpoint, ok := readRTNOutboundProviderConfigString(p.ConfigurationMap, "endpoint"); !ok || endpoint == "" {
		return database.NewValidationError("the outbound RTN provider endpoint is missing")
	}
	if err := validateEbicsRTNOutboundProviderRefs(db, p); err != nil {
		return err
	}

	return validateEbicsRTNOutboundProviderUniqueness(db, p)
}

func (p *EbicsRTNOutboundProvider) hydrateConfiguration() error {
	if p.ConfigurationMap != nil {
		serialized, err := serializeStringMap(p.ConfigurationMap)
		if err != nil {
			return fmt.Errorf("failed to serialize outbound RTN provider configuration: %w", err)
		}

		p.Configuration = serialized
	} else if p.Configuration == "" {
		p.Configuration = emptyJSONObject
	}

	config, err := deserializeStringMap(p.Configuration)
	if err != nil {
		return fmt.Errorf("failed to deserialize outbound RTN provider configuration: %w", err)
	}

	p.ConfigurationMap = config

	return nil
}

func (p *EbicsRTNOutboundProvider) AfterRead(database.ReadAccess) error {
	config, err := deserializeStringMap(p.Configuration)
	if err != nil {
		return fmt.Errorf("failed to deserialize outbound RTN provider configuration after read: %w", err)
	}

	p.ConfigurationMap = config

	return nil
}

func (p *EbicsRTNOutboundProvider) AfterInsert(db database.Access) error { return p.AfterRead(db) }
func (p *EbicsRTNOutboundProvider) AfterUpdate(db database.Access) error { return p.AfterRead(db) }

func validateEbicsRTNOutboundTransport(value string) error {
	switch value {
	case ebicsRTNOutboundTransportWSS:
		return nil
	case "":
		return database.NewValidationError("the outbound RTN provider transport is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported outbound RTN transport", value)
	}
}

func validateEbicsRTNOutboundProviderRefs(db database.Access, provider *EbicsRTNOutboundProvider) error {
	var subscriber EbicsSubscriber
	if err := db.Get(&subscriber, "id=?", provider.EbicsSubscriberID).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf(
				"the EBICS subscriber %d does not exist", provider.EbicsSubscriberID)
		}

		return fmt.Errorf("failed to retrieve EBICS subscriber for outbound RTN provider: %w", err)
	}

	return nil
}

func validateEbicsRTNOutboundProviderUniqueness(db database.Access, provider *EbicsRTNOutboundProvider) error {
	count, err := db.Count(provider).Where(
		"id<>? AND owner=? AND name=?",
		provider.ID, provider.Owner, provider.Name,
	).Run()
	if err != nil {
		return fmt.Errorf("failed to check duplicate outbound RTN providers: %w", err)
	}

	if count != 0 {
		return database.NewValidationErrorf("an outbound RTN provider named %q already exists", provider.Name)
	}

	return nil
}

func readRTNOutboundProviderConfigString(config map[string]any, key string) (string, bool) {
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
