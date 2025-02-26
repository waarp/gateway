package migrations

import "fmt"

func ver0_12_0AddCryptoKeysUp(db Actions) error {
	if err := db.CreateTable("crypto_keys", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "name", Type: Varchar(100), NotNull: true},
			{Name: "type", Type: Varchar(50), NotNull: true},
			{Name: "key", Type: Text{}, NotNull: true},
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
