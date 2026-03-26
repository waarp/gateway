package model

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

const (
	ebicsTransportHTTPS = "https"
	ebicsVersionH004    = "H004"
	ebicsVersionH005    = "H005"
)

type EbicsHost struct {
	ID    int64  `xorm:"<- id AUTOINCR"`
	Owner string `xorm:"owner"`

	Name        string `xorm:"name"`
	HostID      string `xorm:"host_id"`
	Description string `xorm:"description"`

	Enabled         bool   `xorm:"enabled"`
	IsServer        bool   `xorm:"is_server"`
	ProtocolVersion string `xorm:"protocol_version"`
	Transport       string `xorm:"transport"`
	DefaultBankURL  string `xorm:"default_bank_url"`

	CreatedAt time.Time `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt time.Time `xorm:"updated_at UPDATED DATETIME(6) UTC"`
}

func (*EbicsHost) TableName() string   { return TableEbicsHosts }
func (*EbicsHost) Appellation() string { return NameEbicsHost }
func (h *EbicsHost) GetID() int64      { return h.ID }

func (h *EbicsHost) BeforeWrite(db database.Access) error {
	h.Owner = conf.GlobalConfig.GatewayName
	h.Name = strings.TrimSpace(h.Name)
	h.HostID = strings.TrimSpace(h.HostID)
	h.Description = strings.TrimSpace(h.Description)
	h.ProtocolVersion = strings.ToUpper(strings.TrimSpace(h.ProtocolVersion))
	h.Transport = strings.ToLower(strings.TrimSpace(h.Transport))
	h.DefaultBankURL = strings.TrimSpace(h.DefaultBankURL)

	if h.Name == "" {
		h.Name = h.HostID
	}

	if h.Name == "" {
		return database.NewValidationError("the EBICS host name cannot be empty")
	}

	if h.HostID == "" {
		return database.NewValidationError("the EBICS host ID is missing")
	}

	if err := validateEbicsProtocolVersion(h.ProtocolVersion); err != nil {
		return err
	}

	if err := validateEbicsTransport(h.Transport); err != nil {
		return err
	}

	if h.DefaultBankURL != "" {
		parsed, err := url.ParseRequestURI(h.DefaultBankURL)
		if err != nil {
			return database.NewValidationErrorf("invalid EBICS bank URL: %v", err)
		}

		if !strings.EqualFold(parsed.Scheme, ebicsTransportHTTPS) {
			return database.NewValidationError("the EBICS bank URL must use https")
		}
	}

	if n, err := db.Count(h).Where("id<>? AND owner=? AND name=?", h.ID, h.Owner, h.Name).Run(); err != nil {
		return fmt.Errorf("failed to check duplicate EBICS hosts by name: %w", err)
	} else if n != 0 {
		return database.NewValidationErrorf("an EBICS host named %q already exists", h.Name)
	}

	if n, err := db.Count(h).Where("id<>? AND owner=? AND host_id=?", h.ID, h.Owner, h.HostID).Run(); err != nil {
		return fmt.Errorf("failed to check duplicate EBICS hosts by host ID: %w", err)
	} else if n != 0 {
		return database.NewValidationErrorf("an EBICS host with host ID %q already exists", h.HostID)
	}

	return nil
}

func validateEbicsProtocolVersion(version string) error {
	switch version {
	case "":
		return database.NewValidationError("the EBICS protocol version is missing")
	case ebicsVersionH004, ebicsVersionH005:
		return nil
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS protocol version", version)
	}
}

func validateEbicsTransport(transport string) error {
	switch transport {
	case "":
		return database.NewValidationError("the EBICS transport is missing")
	case ebicsTransportHTTPS:
		return nil
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS transport", transport)
	}
}
