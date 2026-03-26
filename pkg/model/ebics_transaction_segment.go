package model

import (
	"fmt"
	"strings"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

const (
	ebicsSegmentStatusPlanned   = "PLANNED"
	ebicsSegmentStatusStored    = "STORED"
	ebicsSegmentStatusSent      = "SENT"
	ebicsSegmentStatusReceived  = "RECEIVED"
	ebicsSegmentStatusCompleted = "COMPLETED"
	ebicsSegmentStatusFailed    = "FAILED"
)

// EbicsTransactionSegment stores the state of a single EBICS transaction segment.
type EbicsTransactionSegment struct {
	ID    int64  `xorm:"<- id AUTOINCR"`
	Owner string `xorm:"owner"`

	EbicsTransactionID int64 `xorm:"ebics_transaction_id"`

	SegmentNumber    int    `xorm:"segment_number"`
	SegmentStatus    string `xorm:"segment_status"`
	PayloadSize      int64  `xorm:"payload_size"`
	Checksum         string `xorm:"checksum"`
	StoredPayloadRef string `xorm:"stored_payload_ref"`
	Metadata         string `xorm:"metadata"`

	CreatedAt time.Time `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt time.Time `xorm:"updated_at UPDATED DATETIME(6) UTC"`

	MetadataMap map[string]any `xorm:"-"`
}

// TableName returns the persistent table name for EBICS transaction segments.
func (*EbicsTransactionSegment) TableName() string { return TableEbicsTransactionSegments }

// Appellation returns the display name used in validation messages.
func (*EbicsTransactionSegment) Appellation() string { return NameEbicsTransactionSegment }

// GetID returns the database identifier of the segment.
func (s *EbicsTransactionSegment) GetID() int64 { return s.ID }

// BeforeWrite normalizes and validates an EBICS transaction segment before persistence.
func (s *EbicsTransactionSegment) BeforeWrite(db database.Access) error {
	s.Owner = conf.GlobalConfig.GatewayName
	s.SegmentStatus = strings.ToUpper(strings.TrimSpace(s.SegmentStatus))
	s.Checksum = strings.TrimSpace(s.Checksum)
	s.StoredPayloadRef = strings.TrimSpace(s.StoredPayloadRef)
	s.Metadata = strings.TrimSpace(s.Metadata)

	if s.EbicsTransactionID == 0 {
		return database.NewValidationError("the EBICS transaction segment reference is missing")
	}

	if s.SegmentNumber <= 0 {
		return database.NewValidationError("the EBICS segment number must be greater than zero")
	}

	if s.PayloadSize < 0 {
		return database.NewValidationError("the EBICS segment payload size cannot be negative")
	}

	if err := validateEbicsSegmentStatus(s.SegmentStatus); err != nil {
		return err
	}

	if s.MetadataMap != nil {
		serialized, err := serializeStringMap(s.MetadataMap)
		if err != nil {
			return fmt.Errorf("failed to serialize EBICS transaction segment metadata: %w", err)
		}

		s.Metadata = serialized
	} else if s.Metadata == "" {
		s.Metadata = emptyJSONObject
	}

	meta, err := deserializeStringMap(s.Metadata)
	if err != nil {
		return fmt.Errorf("failed to deserialize EBICS transaction segment metadata: %w", err)
	}

	s.MetadataMap = meta

	var tx EbicsTransaction
	if err = db.Get(&tx, "id=?", s.EbicsTransactionID).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf(
				"the EBICS transaction %d does not exist", s.EbicsTransactionID)
		}

		return fmt.Errorf("failed to retrieve EBICS transaction: %w", err)
	}

	if n, errCount := db.Count(s).Where(
		"id<>? AND owner=? AND ebics_transaction_id=? AND segment_number=?",
		s.ID, s.Owner, s.EbicsTransactionID, s.SegmentNumber,
	).Run(); errCount != nil {
		return fmt.Errorf("failed to check duplicate EBICS transaction segments: %w", errCount)
	} else if n != 0 {
		return database.NewValidationErrorf(
			"an EBICS transaction segment already exists for segment %d", s.SegmentNumber)
	}

	return nil
}

// AfterRead hydrates the transient metadata map after a database read.
func (s *EbicsTransactionSegment) AfterRead(database.ReadAccess) error {
	meta, err := deserializeStringMap(s.Metadata)
	if err != nil {
		return fmt.Errorf("failed to deserialize EBICS transaction segment metadata after read: %w", err)
	}

	s.MetadataMap = meta

	return nil
}

// AfterInsert refreshes transient state after insertion.
func (s *EbicsTransactionSegment) AfterInsert(db database.Access) error {
	return s.AfterRead(db)
}

// AfterUpdate refreshes transient state after update.
func (s *EbicsTransactionSegment) AfterUpdate(db database.Access) error {
	return s.AfterRead(db)
}

func validateEbicsSegmentStatus(value string) error {
	switch value {
	case ebicsSegmentStatusPlanned, ebicsSegmentStatusStored, ebicsSegmentStatusSent,
		ebicsSegmentStatusReceived, ebicsSegmentStatusCompleted, ebicsSegmentStatusFailed:
		return nil
	case "":
		return database.NewValidationError("the EBICS segment status is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS segment status", value)
	}
}

// EbicsTransactionSegmentStatusStoredForRuntime exposes the stored segment status.
func EbicsTransactionSegmentStatusStoredForRuntime() string {
	return ebicsSegmentStatusStored
}

// EbicsTransactionSegmentStatusCompletedForRuntime exposes the completed segment status.
func EbicsTransactionSegmentStatusCompletedForRuntime() string {
	return ebicsSegmentStatusCompleted
}
