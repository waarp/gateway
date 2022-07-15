package migrations

import (
	"fmt"

	"code.waarp.fr/lib/migration"
)

type ver0_7_0AddLocalAgentEnabledColumn struct{}

func (ver0_7_0AddLocalAgentEnabledColumn) Up(db migration.Actions) error {
	if err := db.AddColumn("local_agents", "enabled", migration.Boolean,
		migration.NotNull, migration.Default(true)); err != nil {
		return fmt.Errorf("failed to add the local agent 'enabled' column: %w", err)
	}

	return nil
}

func (ver0_7_0AddLocalAgentEnabledColumn) Down(db migration.Actions) error {
	if err := db.DropColumn("local_agents", "enabled"); err != nil {
		return fmt.Errorf("failed to drop the local agent 'enabled' column: %w", err)
	}

	return nil
}
