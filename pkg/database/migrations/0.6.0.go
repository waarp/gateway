package migrations

import (
	"fmt"
)

type ver0_6_0AddTransferInfoIsHistory struct{}

func (ver0_6_0AddTransferInfoIsHistory) Up(db Actions) error {
	if err := db.AlterTable("transfer_info",
		AddColumn{Name: "is_history", Type: Boolean{}, NotNull: true, Default: true},
	); err != nil {
		return fmt.Errorf("failed to add transfer info 'is_history' column: %w", err)
	}

	return nil
}

func (ver0_6_0AddTransferInfoIsHistory) Down(db Actions) error {
	if err := db.AlterTable("transfer_info", DropColumn{Name: "is_history"}); err != nil {
		return fmt.Errorf("failed to drop transfer info 'is_history' column: %w", err)
	}

	return nil
}
