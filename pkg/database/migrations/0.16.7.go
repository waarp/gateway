package migrations

import "fmt"

func ver0_16_7AddEbicsServerReportingSetsUp(db Actions) error {
	if err := db.CreateTable("ebics_server_reporting_sets", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "owner", Type: Varchar(50), NotNull: true},
			{Name: "name", Type: Varchar(100), NotNull: true},
			{Name: "description", Type: Text{}},
			{Name: "ebics_host_id", Type: BigInt{}, NotNull: true},
			{Name: "ebics_subscriber_id", Type: BigInt{}, NotNull: true},
			{Name: "source_order_type", Type: Varchar(20), NotNull: true},
			{Name: "version_tag", Type: Varchar(100), NotNull: true},
			{Name: "status", Type: Varchar(20), NotNull: true},
			{Name: "published_at", Type: DateTime{}},
			{Name: "created_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
			{Name: "updated_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
		},
		PrimaryKey: &PrimaryKey{Name: "ebics_server_reporting_sets_pkey", Cols: []string{"id"}},
		ForeignKeys: []ForeignKey{
			{
				Name:    "ebics_server_reporting_sets_host_id_fkey",
				Cols:    []string{"ebics_host_id"},
				RefTbl:  "ebics_hosts",
				RefCols: []string{"id"},
			},
			{
				Name:    "ebics_server_reporting_sets_subscriber_id_fkey",
				Cols:    []string{"ebics_subscriber_id"},
				RefTbl:  "ebics_subscribers",
				RefCols: []string{"id"},
			},
		},
	}); err != nil {
		return fmt.Errorf(`failed to create "ebics_server_reporting_sets": %w`, err)
	}

	if err := db.CreateIndex(&Index{
		Name: "ebics_server_reporting_sets_source_idx",
		On:   "ebics_server_reporting_sets",
		Cols: []string{"owner", "ebics_host_id", "ebics_subscriber_id", "source_order_type", "status"},
	}); err != nil {
		return fmt.Errorf(`failed to create index on "ebics_server_reporting_sets": %w`, err)
	}

	if err := db.CreateTable("ebics_server_reporting_items", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "owner", Type: Varchar(50), NotNull: true},
			{Name: "server_reporting_set_id", Type: BigInt{}, NotNull: true},
			{Name: "item_key", Type: Varchar(150), NotNull: true},
			{Name: "order_id", Type: Varchar(100)},
			{Name: "service_name", Type: Varchar(30)},
			{Name: "service_option", Type: Varchar(30)},
			{Name: "scope", Type: Varchar(30)},
			{Name: "msg_name", Type: Varchar(100)},
			{Name: "container_type", Type: Varchar(30)},
			{Name: "is_enabled", Type: Boolean{}, NotNull: true, Default: true},
			{Name: "response_payload", Type: Blob{}, NotNull: true},
			{Name: "original_payload", Type: Blob{}},
			{Name: "created_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
			{Name: "updated_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
		},
		PrimaryKey: &PrimaryKey{Name: "ebics_server_reporting_items_pkey", Cols: []string{"id"}},
		ForeignKeys: []ForeignKey{
			{
				Name:    "ebics_server_reporting_items_set_id_fkey",
				Cols:    []string{"server_reporting_set_id"},
				RefTbl:  "ebics_server_reporting_sets",
				RefCols: []string{"id"},
			},
		},
	}); err != nil {
		return fmt.Errorf(`failed to create "ebics_server_reporting_items": %w`, err)
	}

	if err := db.CreateIndex(&Index{
		Name: "ebics_server_reporting_items_set_idx",
		On:   "ebics_server_reporting_items",
		Cols: []string{"server_reporting_set_id", "order_id", "service_name", "msg_name"},
	}); err != nil {
		return fmt.Errorf(`failed to create index on "ebics_server_reporting_items": %w`, err)
	}

	return nil
}

func ver0_16_7AddEbicsServerReportingSetsDown(db Actions) error {
	if err := db.DropTable("ebics_server_reporting_items"); err != nil {
		return fmt.Errorf(`failed to drop "ebics_server_reporting_items": %w`, err)
	}
	if err := db.DropTable("ebics_server_reporting_sets"); err != nil {
		return fmt.Errorf(`failed to drop "ebics_server_reporting_sets": %w`, err)
	}

	return nil
}
