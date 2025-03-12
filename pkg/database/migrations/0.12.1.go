package migrations

import "fmt"

func ver0_12_1AddCryptoKeysOwnerUp(db Actions) error {
	if err := db.AlterTable("crypto_keys",
		RenameColumn{OldName: quote(db, "key"), NewName: "value"},
		DropConstraint{Name: "unique_crypto_keys"},
		AddColumn{Name: "owner", Type: Varchar(100)},
	); err != nil {
		return fmt.Errorf(`failed to add the "crypto_keys.owner" column: %w`, err)
	}

	if err := db.Exec(`INSERT INTO crypto_keys(owner, name, type, value)
		SELECT * FROM
			(SELECT DISTINCT owner FROM users) a INNER JOIN
			(SELECT name, type, value FROM crypto_keys) b ON true`,
	); err != nil {
		return fmt.Errorf(`failed to duplicate crypto keys: %w`, err)
	}

	if err := db.Exec(`DELETE FROM crypto_keys WHERE owner IS NULL`); err != nil {
		return fmt.Errorf(`failed to remove the old crypto keys: %w`, err)
	}

	if err := db.AlterTable("crypto_keys",
		AddUnique{Name: "unique_crypto_keys_owner", Cols: []string{"owner", "name"}},
		AlterColumn{Name: "owner", Type: Varchar(100), NotNull: true},
	); err != nil {
		return fmt.Errorf(`failed to alter the "crypto_keys.owner" column: %w`, err)
	}

	return nil
}

func ver0_12_1AddCryptoKeysOwnerDown(db Actions) error {
	if err := db.Exec(`DELETE FROM crypto_keys WHERE 
		owner<>(SELECT owner FROM crypto_keys LIMIT 1)`); err != nil {
		return fmt.Errorf(`failed to remove the duplicate crypto keys: %w`, err)
	}

	if err := db.AlterTable("crypto_keys",
		DropConstraint{Name: "unique_crypto_keys_owner"},
	); err != nil {
		return fmt.Errorf(`failed to drop the "unique_crypto_keys_owner" column: %w`, err)
	}

	if err := db.AlterTable("crypto_keys",
		DropColumn{Name: "owner"},
		AddUnique{Name: "unique_crypto_keys", Cols: []string{"name"}},
	); err != nil {
		return fmt.Errorf(`failed to drop the "crypto_keys.owner" column: %w`, err)
	}

	if err := db.AlterTable("crypto_keys",
		RenameColumn{OldName: "value", NewName: quote(db, "key")},
	); err != nil {
		return fmt.Errorf(`failed to rename the "crypto_keys.key" column: %w`, err)
	}

	return nil
}
