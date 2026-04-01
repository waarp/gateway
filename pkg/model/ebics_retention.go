package model

import (
	"fmt"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

// PurgeEbicsNoncesBefore removes expired EBICS nonces strictly older than the cutoff.
func PurgeEbicsNoncesBefore(db database.Access, cutoff time.Time) error {
	if err := db.DeleteAll(&EbicsNonce{}).
		Where("owner=? AND expires_at<?", conf.GlobalConfig.GatewayName, cutoff.UTC()).
		Run(); err != nil {
		return fmt.Errorf("purge EBICS nonces: %w", err)
	}

	return nil
}

// PurgeEbicsTransactionsBefore removes EBICS transactions strictly older than the cutoff.
func PurgeEbicsTransactionsBefore(db database.Access, cutoff time.Time) error {
	if err := db.DeleteAll(&EbicsTransaction{}).
		Where("owner=? AND updated_at<?", conf.GlobalConfig.GatewayName, cutoff.UTC()).
		Run(); err != nil {
		return fmt.Errorf("purge EBICS transactions: %w", err)
	}

	return nil
}

// PurgeEbicsRTNEventsBefore removes terminal RTN events strictly older than the cutoff.
func PurgeEbicsRTNEventsBefore(db database.Access, cutoff time.Time) error {
	if err := db.DeleteAll(&EbicsRTNEvent{}).
		Where(
			"owner=? AND status IN (?, ?, ?, ?) AND updated_at<?",
			conf.GlobalConfig.GatewayName,
			ebicsRTNEventStatusDuplicate,
			ebicsRTNEventStatusProcessed,
			ebicsRTNEventStatusQuarantined,
			ebicsRTNEventStatusFailed,
			cutoff.UTC(),
		).
		Run(); err != nil {
		return fmt.Errorf("purge EBICS RTN events: %w", err)
	}

	return nil
}
