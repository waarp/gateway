package migrations

import (
	"fmt"

	"code.waarp.fr/lib/migration"
)

func ver0_4_2RemoveHistoryRemoteIDUniqueUp(db migration.Actions) error {
	if err := db.DropIndex(quote(db, "UQE_transfer_history_histRemID"),
		"transfer_history"); err != nil {
		return fmt.Errorf("failed to drop the history id index: %w", err)
	}

	return nil
}

func ver0_4_2RemoveHistoryRemoteIDUniqueDown(db migration.Actions) error {
	if err := db.CreateIndex(&Index{
		Name: quote(db, "UQE_transfer_history_histRemID"), Unique: true,
		On: "transfer_history", Cols: []string{"remote_transfer_id", "account", "agent"},
	}); err != nil {
		return fmt.Errorf("failed to restore the history id index: %w", err)
	}

	return nil
}
