package migrations

import "fmt"

func ver0_16_1AddEbicsStandardBTFCatalogsUp(db Actions) error {
	if err := ver0_16_1CreateEbicsStandardBTFCatalogs(db); err != nil {
		return err
	}

	return ver0_16_1CreateEbicsStandardBTFEntries(db)
}

func ver0_16_1AddEbicsStandardBTFCatalogsDown(db Actions) error {
	for _, table := range []string{
		"ebics_standard_btf_entries",
		"ebics_standard_btf_catalogs",
	} {
		if err := db.DropTable(table); err != nil {
			return fmt.Errorf("failed to drop %q: %w", table, err)
		}
	}

	return nil
}

func ver0_16_1CreateEbicsStandardBTFCatalogs(db Actions) error {
	err := db.CreateTable("ebics_standard_btf_catalogs", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "owner", Type: Varchar(50), NotNull: true},
			{Name: "name", Type: Varchar(100), NotNull: true},
			{Name: "scope", Type: Varchar(10), NotNull: true},
			{Name: "catalog_version", Type: Varchar(100), NotNull: true},
			{Name: "source_type", Type: Varchar(30), NotNull: true},
			{Name: "source_ref", Type: Text{}, NotNull: true, Default: ""},
			{Name: "status", Type: Varchar(30), NotNull: true},
			{Name: "seed_checksum", Type: Varchar(255), NotNull: true, Default: ""},
			{Name: "created_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
			{Name: "updated_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
		},
		PrimaryKey: &PrimaryKey{Name: "ebics_standard_btf_catalogs_pkey", Cols: []string{"id"}},
		Uniques: []Unique{{
			Name: "unique_ebics_standard_btf_catalog_identity",
			Cols: []string{"owner", "name", "scope", "catalog_version"},
		}},
	})
	if err != nil {
		return fmt.Errorf(`failed to create "ebics_standard_btf_catalogs": %w`, err)
	}

	return nil
}

func ver0_16_1CreateEbicsStandardBTFEntries(db Actions) error {
	err := db.CreateTable("ebics_standard_btf_entries", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "owner", Type: Varchar(50), NotNull: true},
			{Name: "catalog_id", Type: BigInt{}, NotNull: true},
			{Name: "entry_key", Type: Varchar(255), NotNull: true},
			{Name: "order_type", Type: Varchar(20), NotNull: true},
			{Name: "direction", Type: Varchar(20), NotNull: true},
			{Name: "service_name", Type: Varchar(50), NotNull: true},
			{Name: "service_option", Type: Varchar(50), NotNull: true, Default: ""},
			{Name: "scope", Type: Varchar(10), NotNull: true},
			{Name: "msg_name", Type: Varchar(100), NotNull: true, Default: ""},
			{Name: "container_type", Type: Varchar(100), NotNull: true, Default: ""},
			{Name: "country_group", Type: Varchar(10), NotNull: true, Default: ""},
			{Name: "is_default_template", Type: Boolean{}, NotNull: true, Default: false},
			{Name: "status", Type: Varchar(30), NotNull: true},
			{Name: "metadata", Type: Text{}, NotNull: true, Default: "{}"},
			{Name: "created_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
			{Name: "updated_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
		},
		PrimaryKey: &PrimaryKey{Name: "ebics_standard_btf_entries_pkey", Cols: []string{"id"}},
		ForeignKeys: []ForeignKey{{
			Name: "ebics_standard_btf_entries_catalog_fkey", Cols: []string{"catalog_id"},
			RefTbl: "ebics_standard_btf_catalogs", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: Cascade,
		}},
		Uniques: []Unique{{
			Name: "unique_ebics_standard_btf_entry_key",
			Cols: []string{"owner", "catalog_id", "entry_key"},
		}},
	})
	if err != nil {
		return fmt.Errorf(`failed to create "ebics_standard_btf_entries": %w`, err)
	}

	return nil
}
