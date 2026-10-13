package migrations

import "fmt"

func ver0_17_0ChangeTransferInfoValueTypeUp(db Actions) error {
	if db.GetDialect() != MySQL {
		return nil
	}

	if err := db.ExecDDL(`ALTER TABLE transfer_info MODIFY COLUMN ` + quote(db, "value") +
		` LONGTEXT NOT NULL DEFAULT 'null'`); err != nil {
		return fmt.Errorf("failed to change transfer info value type: %w", err)
	}

	return nil
}

func ver0_17_0ChangeTransferInfoValueTypeDown(db Actions) error {
	if db.GetDialect() != MySQL {
		return nil
	}

	if err := db.ExecDDL(`ALTER TABLE transfer_info MODIFY COLUMN ` + quote(db, "value") +
		` TEXT NOT NULL DEFAULT 'null'`); err != nil {
		return fmt.Errorf("failed to change transfer info value type: %w", err)
	}

	return nil
}
