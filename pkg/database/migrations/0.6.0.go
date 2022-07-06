package migrations

import (
	"fmt"

	"code.waarp.fr/lib/migration"
)

type ver0_6_0AddTransferInfoIsHistory struct{}

func (ver0_6_0AddTransferInfoIsHistory) Up(db migration.Actions) error {
	if err := db.AddColumn("transfer_info", "is_history", migration.Boolean,
		migration.NotNull, migration.Default(true)); err != nil {
		return fmt.Errorf("failed to add transfer info 'is_history' column: %w", err)
	}

	return nil
}

func (ver0_6_0AddTransferInfoIsHistory) Down(db migration.Actions) error {
	if err := db.DropColumn("transfer_info", "is_history"); err != nil {
		return fmt.Errorf("failed to drop transfer info 'is_history' column: %w", err)
	}

	return nil
}
