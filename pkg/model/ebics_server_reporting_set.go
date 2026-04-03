package model

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

const (
	ebicsServerReportingSetStatusActive     = "ACTIVE"
	ebicsServerReportingSetStatusDraft      = "DRAFT"
	ebicsServerReportingSetStatusSuperseded = "SUPERSEDED"
)

type EbicsServerReportingSet struct {
	ID    int64  `xorm:"<- id AUTOINCR"`
	Owner string `xorm:"owner"`

	Name        string `xorm:"name"`
	Description string `xorm:"description"`

	EbicsHostID       int64         `xorm:"ebics_host_id"`
	EbicsSubscriberID sql.NullInt64 `xorm:"ebics_subscriber_id"`

	SourceOrderType string `xorm:"source_order_type"`
	VersionTag      string `xorm:"version_tag"`
	Status          string `xorm:"status"`

	PublishedAt time.Time `xorm:"published_at DATETIME(6) UTC"`
	CreatedAt   time.Time `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt   time.Time `xorm:"updated_at UPDATED DATETIME(6) UTC"`
}

func (*EbicsServerReportingSet) TableName() string   { return TableEbicsServerReportingSets }
func (*EbicsServerReportingSet) Appellation() string { return NameEbicsServerReportingSet }
func (s *EbicsServerReportingSet) GetID() int64      { return s.ID }

func (s *EbicsServerReportingSet) BeforeWrite(db database.Access) error {
	s.Owner = conf.GlobalConfig.GatewayName
	s.Name = strings.TrimSpace(s.Name)
	s.Description = strings.TrimSpace(s.Description)
	s.SourceOrderType = strings.ToUpper(strings.TrimSpace(s.SourceOrderType))
	s.VersionTag = strings.TrimSpace(s.VersionTag)
	s.Status = strings.ToUpper(strings.TrimSpace(s.Status))

	if s.EbicsHostID == 0 {
		return database.NewValidationError("the EBICS host reference is missing")
	}
	if !s.EbicsSubscriberID.Valid {
		return database.NewValidationError("the EBICS server reporting set requires a subscriber")
	}
	if err := validateEbicsServerReportingSourceOrderType(s.SourceOrderType); err != nil {
		return err
	}
	if err := validateEbicsServerReportingSetStatus(s.Status); err != nil {
		return err
	}

	if s.Name == "" {
		s.Name = defaultEbicsServerReportingSetName(s)
	}
	if s.VersionTag == "" {
		s.VersionTag = time.Now().UTC().Format(time.RFC3339Nano)
	}
	if s.Status == ebicsServerReportingSetStatusActive && s.PublishedAt.IsZero() {
		s.PublishedAt = time.Now().UTC()
	}

	var host EbicsHost
	if err := db.Get(&host, "id=?", s.EbicsHostID).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf("the EBICS host %d does not exist", s.EbicsHostID)
		}

		return fmt.Errorf("failed to retrieve EBICS host: %w", err)
	}

	var subscriber EbicsSubscriber
	if err := db.Get(&subscriber, "id=?", s.EbicsSubscriberID.Int64).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf(
				"the EBICS subscriber %d does not exist", s.EbicsSubscriberID.Int64)
		}

		return fmt.Errorf("failed to retrieve EBICS subscriber: %w", err)
	}
	if subscriber.EbicsHostID != s.EbicsHostID {
		return database.NewValidationError(
			"the EBICS server reporting subscriber does not belong to the selected host")
	}

	n, err := db.Count(s).Where("id<>? AND owner=? AND name=?", s.ID, s.Owner, s.Name).Run()
	if err != nil {
		return fmt.Errorf("failed to check duplicate EBICS server reporting sets: %w", err)
	}
	if n != 0 {
		return database.NewValidationErrorf("an EBICS server reporting set named %q already exists", s.Name)
	}

	return nil
}

func validateEbicsServerReportingSetStatus(status string) error {
	switch status {
	case ebicsServerReportingSetStatusActive, ebicsServerReportingSetStatusDraft,
		ebicsServerReportingSetStatusSuperseded:
		return nil
	case "":
		return database.NewValidationError("the EBICS server reporting set status is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS server reporting set status", status)
	}
}

func validateEbicsServerReportingSourceOrderType(orderType string) error {
	switch NormalizeEbicsOrderType(orderType) {
	case "HVD", "HVE", "HVU", "HVZ", "HVT", "HAC", "HVS":
		return nil
	case "":
		return database.NewValidationError("the EBICS server reporting source order type is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS server reporting source order type", orderType)
	}
}

func defaultEbicsServerReportingSetName(set *EbicsServerReportingSet) string {
	return strings.ToLower(
		fmt.Sprintf("%s-%d-subscriber-%d", set.SourceOrderType, set.EbicsHostID, set.EbicsSubscriberID.Int64),
	)
}
