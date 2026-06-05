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

func ver0_16_0AddAckTrackingUp(db Actions) error {
	if err := db.CreateTable("ack_tracking", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "transfer_id", Type: BigInt{}, NotNull: true},
			{Name: "remote_id", Type: Text{}},
			{Name: "is_send", Type: Boolean{}, NotNull: true},
			{Name: "state", Type: Varchar(20), NotNull: true},
			{Name: "partner", Type: Text{}},
			{Name: "account", Type: Text{}},
			{Name: "origin", Type: Text{}},
			{Name: "message", Type: Text{}},
			{Name: "customer_id", Type: Text{}},
			{Name: "bank_id", Type: Text{}},
			{Name: "created_at", Type: DateTime{}, NotNull: true},
			{Name: "updated_at", Type: DateTime{}, NotNull: true},
		},
		PrimaryKey: &PrimaryKey{Name: "ack_tracking_pkey", Cols: []string{"id"}},
		Uniques: []Unique{
			{Name: "unique_ack_transfer", Cols: []string{"transfer_id"}},
		},
	}); err != nil {
		return fmt.Errorf(`failed to create "ack_tracking" table: %w`, err)
	}

	return nil
}

func ver0_16_0AddAckTrackingDown(db Actions) error {
	if err := db.DropTable("ack_tracking"); err != nil {
		return fmt.Errorf(`failed to drop "ack_tracking" table: %w`, err)
	}

	return nil
}
