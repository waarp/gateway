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
	ebicsContractViewStatusActive     = "ACTIVE"
	ebicsContractViewStatusSuperseded = "SUPERSEDED"
	ebicsContractViewStatusStale      = "STALE"

	ebicsContractSourceOrderHPD = "HPD"
	ebicsContractSourceOrderHKD = "HKD"
	ebicsContractSourceOrderHTD = "HTD"
	ebicsContractSourceOrderHAA = "HAA"
)

type EbicsContractView struct {
	ID    int64  `xorm:"<- id AUTOINCR"`
	Owner string `xorm:"owner"`

	EbicsHostID       int64         `xorm:"ebics_host_id"`
	EbicsSubscriberID sql.NullInt64 `xorm:"ebics_subscriber_id"`

	SourceOrderType   string        `xorm:"source_order_type"`
	SourceOperationID sql.NullInt64 `xorm:"source_operation_id"`
	VersionTag        string        `xorm:"version_tag"`
	Status            string        `xorm:"status"`

	FetchedAt time.Time `xorm:"fetched_at DATETIME(6) UTC"`
	CreatedAt time.Time `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt time.Time `xorm:"updated_at UPDATED DATETIME(6) UTC"`
}

func (*EbicsContractView) TableName() string   { return TableEbicsContractViews }
func (*EbicsContractView) Appellation() string { return NameEbicsContractView }
func (v *EbicsContractView) GetID() int64      { return v.ID }

func (v *EbicsContractView) BeforeWrite(db database.Access) error {
	v.Owner = conf.GlobalConfig.GatewayName
	v.SourceOrderType = strings.ToUpper(strings.TrimSpace(v.SourceOrderType))
	v.VersionTag = strings.TrimSpace(v.VersionTag)
	v.Status = strings.ToUpper(strings.TrimSpace(v.Status))

	if v.EbicsHostID == 0 {
		return database.NewValidationError("the EBICS host reference is missing")
	}

	if err := validateEbicsContractSourceOrderType(v.SourceOrderType); err != nil {
		return err
	}

	if err := validateEbicsContractViewStatus(v.Status); err != nil {
		return err
	}

	if v.FetchedAt.IsZero() {
		v.FetchedAt = time.Now().UTC()
	}

	var host EbicsHost
	if err := db.Get(&host, "id=?", v.EbicsHostID).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf("the EBICS host %d does not exist", v.EbicsHostID)
		}

		return fmt.Errorf("failed to retrieve EBICS host: %w", err)
	}

	if v.EbicsSubscriberID.Valid {
		var subscriber EbicsSubscriber
		if err := db.Get(&subscriber, "id=?", v.EbicsSubscriberID.Int64).Run(); err != nil {
			if database.IsNotFound(err) {
				return database.NewValidationErrorf(
					"the EBICS subscriber %d does not exist", v.EbicsSubscriberID.Int64)
			}

			return fmt.Errorf("failed to retrieve EBICS subscriber: %w", err)
		}

		if subscriber.EbicsHostID != v.EbicsHostID {
			return database.NewValidationError(
				"the EBICS contract view subscriber does not belong to the selected host")
		}
	}

	return nil
}

func validateEbicsContractViewStatus(status string) error {
	switch status {
	case ebicsContractViewStatusActive, ebicsContractViewStatusSuperseded, ebicsContractViewStatusStale:
		return nil
	case "":
		return database.NewValidationError("the EBICS contract view status is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS contract view status", status)
	}
}

func validateEbicsContractSourceOrderType(orderType string) error {
	switch orderType {
	case ebicsContractSourceOrderHPD, ebicsContractSourceOrderHKD,
		ebicsContractSourceOrderHTD, ebicsContractSourceOrderHAA:
		return nil
	case "":
		return database.NewValidationError("the EBICS contract source order type is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS contract source order type", orderType)
	}
}
