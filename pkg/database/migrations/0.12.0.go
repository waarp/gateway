package migrations

import "fmt"

func ver0_12_0AddPGPKeysUp(db Actions) error {
	if err := db.CreateTable("pgp_keys", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "name", Type: Text{}, NotNull: true},
			{Name: "private_key", Type: Text{}, NotNull: true},
			{Name: "public_key", Type: Text{}, NotNull: true},
		},
		PrimaryKey: &PrimaryKey{Name: "pgp_keys_pkey", Cols: []string{"id"}},
		Uniques: []Unique{
			{Name: "unique_snmp_monitor", Cols: []string{"name"}},
		},
	}); err != nil {
		return fmt.Errorf(`failed to create the "pgp_keys" table: %w`, err)
	}

	return nil
}

func ver0_12_0AddPGPKeysDown(db Actions) error {
	if err := db.DropTable("pgp_keys"); err != nil {
		return fmt.Errorf(`failed to drop the "pgp_keys" table: %w`, err)
	}

	return nil
}
