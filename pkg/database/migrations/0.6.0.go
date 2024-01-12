package migrations

import (
	"fmt"
)

func ver0_6_0AddTransferInfoIsHistoryUp(db Actions) error {
	if err := db.AlterTable("transfer_info",
		AddColumn{Name: "is_history", Type: Boolean{}, NotNull: true, Default: true},
	); err != nil {
		return fmt.Errorf("failed to add transfer info 'is_history' column: %w", err)
	}

	return nil
}

func ver0_6_0AddTransferInfoIsHistoryDown(db Actions) error {
	if err := db.AlterTable("transfer_info", DropColumn{Name: "is_history"}); err != nil {
		return fmt.Errorf("failed to drop transfer info 'is_history' column: %w", err)
	}

	return nil
}
