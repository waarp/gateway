package migrations

import "fmt"

func ver0_12_0AddCryptoKeysUp(db Actions) error {
	if err := db.CreateTable("crypto_keys", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "name", Type: Varchar(100), NotNull: true},
			{Name: "type", Type: Varchar(50), NotNull: true},
			{Name: quote(db, "key"), Type: Text{}, NotNull: true},
		},
		PrimaryKey: &PrimaryKey{Name: "crypto_keys_pkey", Cols: []string{"id"}},
		Uniques: []Unique{
			{Name: "unique_crypto_keys", Cols: []string{"name"}},
		},
	}); err != nil {
		return fmt.Errorf(`failed to create the "crypto_keys" table: %w`, err)
	}

	return nil
}

func ver0_12_0AddCryptoKeysDown(db Actions) error {
	if err := db.DropTable("crypto_keys"); err != nil {
		return fmt.Errorf(`failed to drop the "crypto_keys" table: %w`, err)
	}

	return nil
}

func ver0_12_0DropRemoteTransferIdUniqueUp(db Actions) error {
	if err := db.AlterTable("transfers",
		DropConstraint{Name: "unique_transfer_local"},
		DropConstraint{Name: "unique_transfer_remote"},
	); err != nil {
		return fmt.Errorf(`failed to drop "transfers" constraint: %w`, err)
	}

	if err := db.AlterTable("transfer_history",
		DropConstraint{Name: "unique_history"},
	); err != nil {
		return fmt.Errorf(`failed to drop "transfer_history" constraints: %w`, err)
	}

	return nil
}

func ver0_12_0DropRemoteTransferIdUniqueDown(db Actions) error {
	if err := db.AlterTable("transfer_history",
		AddUnique{Name: "unique_history", Cols: []string{
			"remote_transfer_id",
			"is_server", "account", "agent",
		}},
	); err != nil {
		return fmt.Errorf(`failed to restore the "transfer_history" constraint: %w`, err)
	}

	if err := db.AlterTable("transfers",
		AddUnique{Name: "unique_transfer_local", Cols: []string{"remote_transfer_id", "local_account_id"}},
		AddUnique{Name: "unique_transfer_remote", Cols: []string{"remote_transfer_id", "remote_account_id"}},
	); err != nil {
		return fmt.Errorf(`failed to restore the "transfers" constraint: %w`, err)
	}

	return nil
}
