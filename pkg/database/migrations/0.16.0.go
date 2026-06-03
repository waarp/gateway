package migrations

import "fmt"

func ver0_16_0AddTaskConditionUp(db Actions) error {
	if err := db.AlterTable("tasks",
		AddColumn{Name: "condition", Type: Text{}, NotNull: true, Default: ""},
	); err != nil {
		return fmt.Errorf(`failed to add "condition" column to "tasks": %w`, err)
	}

	return nil
}

func ver0_16_0AddTaskConditionDown(db Actions) error {
	if err := db.AlterTable("tasks",
		DropColumn{Name: "condition"},
	); err != nil {
		return fmt.Errorf(`failed to drop "condition" column from "tasks": %w`, err)
	}

	return nil
}
