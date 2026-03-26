package model

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

type EbicsSubscriber struct {
	ID    int64  `xorm:"<- id AUTOINCR"`
	Owner string `xorm:"owner"`

	EbicsHostID int64 `xorm:"ebics_host_id"`

	Name      string `xorm:"name"`
	PartnerID string `xorm:"partner_id"`
	UserID    string `xorm:"user_id"`
	SystemID  string `xorm:"system_id"`

	TransportURL             string `xorm:"transport_url"`
	Enabled                  bool   `xorm:"enabled"`
	DefaultOrderDataEncoding string `xorm:"default_order_data_encoding"`

	CreatedAt time.Time `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt time.Time `xorm:"updated_at UPDATED DATETIME(6) UTC"`
}

func (*EbicsSubscriber) TableName() string   { return TableEbicsSubscribers }
func (*EbicsSubscriber) Appellation() string { return NameEbicsSubscriber }
func (s *EbicsSubscriber) GetID() int64      { return s.ID }

func (s *EbicsSubscriber) BeforeWrite(db database.Access) error {
	s.Owner = conf.GlobalConfig.GatewayName
	s.Name = strings.TrimSpace(s.Name)
	s.PartnerID = strings.TrimSpace(s.PartnerID)
	s.UserID = strings.TrimSpace(s.UserID)
	s.SystemID = strings.TrimSpace(s.SystemID)
	s.TransportURL = strings.TrimSpace(s.TransportURL)
	s.DefaultOrderDataEncoding = strings.TrimSpace(s.DefaultOrderDataEncoding)

	if s.EbicsHostID == 0 {
		return database.NewValidationError("the EBICS host reference is missing")
	}

	if s.PartnerID == "" {
		return database.NewValidationError("the EBICS partner ID is missing")
	}

	if s.UserID == "" {
		return database.NewValidationError("the EBICS user ID is missing")
	}

	if s.Name == "" {
		s.Name = s.PartnerID + ":" + s.UserID
	}

	if s.TransportURL != "" {
		if _, err := url.ParseRequestURI(s.TransportURL); err != nil {
			return database.NewValidationErrorf("invalid EBICS transport URL: %v", err)
		}
	}

	var host EbicsHost
	if err := db.Get(&host, "id=?", s.EbicsHostID).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf("the EBICS host %d does not exist", s.EbicsHostID)
		}

		return fmt.Errorf("failed to retrieve EBICS host: %w", err)
	}

	if n, err := db.Count(s).Where(
		"id<>? AND owner=? AND ebics_host_id=? AND partner_id=? AND user_id=?",
		s.ID, s.Owner, s.EbicsHostID, s.PartnerID, s.UserID,
	).Run(); err != nil {
		return fmt.Errorf("failed to check duplicate EBICS subscribers: %w", err)
	} else if n != 0 {
		return database.NewValidationErrorf(
			"an EBICS subscriber already exists for partner %q and user %q", s.PartnerID, s.UserID)
	}

	if n, err := db.Count(s).Where("id<>? AND owner=? AND name=?", s.ID, s.Owner, s.Name).Run(); err != nil {
		return fmt.Errorf("failed to check duplicate EBICS subscribers by name: %w", err)
	} else if n != 0 {
		return database.NewValidationErrorf("an EBICS subscriber named %q already exists", s.Name)
	}

	return nil
}
