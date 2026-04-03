package migrations

import "fmt"

func ver0_16_8AddEbicsOutboundRTNTablesUp(db Actions) error {
	if err := db.CreateTable("ebics_rtn_outbound_providers", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "owner", Type: Varchar(50), NotNull: true},
			{Name: "name", Type: Varchar(100), NotNull: true},
			{Name: "transport", Type: Varchar(20), NotNull: true},
			{Name: "enabled", Type: Boolean{}, NotNull: true, Default: true},
			{Name: "ebics_subscriber_id", Type: BigInt{}, NotNull: true},
			{Name: "configuration", Type: Text{}, NotNull: true},
			{Name: "last_connection_at", Type: DateTime{}},
			{Name: "last_error", Type: Text{}},
			{Name: "created_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
			{Name: "updated_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
		},
		PrimaryKey: &PrimaryKey{Name: "ebics_rtn_outbound_providers_pkey", Cols: []string{"id"}},
		ForeignKeys: []ForeignKey{
			{
				Name:    "ebics_rtn_outbound_providers_subscriber_id_fkey",
				Cols:    []string{"ebics_subscriber_id"},
				RefTbl:  "ebics_subscribers",
				RefCols: []string{"id"},
			},
		},
	}); err != nil {
		return fmt.Errorf(`failed to create "ebics_rtn_outbound_providers": %w`, err)
	}

	if err := db.CreateIndex(&Index{
		Name: "ebics_rtn_outbound_providers_name_idx",
		On:   "ebics_rtn_outbound_providers",
		Cols: []string{"owner", "name"},
	}); err != nil {
		return fmt.Errorf(`failed to create index on "ebics_rtn_outbound_providers": %w`, err)
	}

	if err := db.CreateTable("ebics_rtn_outbound_notifications", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "owner", Type: Varchar(50), NotNull: true},
			{Name: "provider_id", Type: BigInt{}, NotNull: true},
			{Name: "event_type", Type: Varchar(40), NotNull: true},
			{Name: "source_order_type", Type: Varchar(20), NotNull: true},
			{Name: "correlation_id", Type: Varchar(150)},
			{Name: "ebics_host_id", Type: BigInt{}, NotNull: true},
			{Name: "ebics_subscriber_id", Type: BigInt{}, NotNull: true},
			{Name: "server_reporting_set_id", Type: BigInt{}},
			{Name: "server_reporting_item_key", Type: Varchar(150)},
			{Name: "payload", Type: Text{}, NotNull: true},
			{Name: "status", Type: Varchar(20), NotNull: true},
			{Name: "attempts", Type: BigInt{}, NotNull: true, Default: 0},
			{Name: "next_retry_at", Type: DateTime{}},
			{Name: "sent_at", Type: DateTime{}},
			{Name: "last_error", Type: Text{}},
			{Name: "created_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
			{Name: "updated_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
		},
		PrimaryKey: &PrimaryKey{Name: "ebics_rtn_outbound_notifications_pkey", Cols: []string{"id"}},
		ForeignKeys: []ForeignKey{
			{
				Name:    "ebics_rtn_outbound_notifications_provider_id_fkey",
				Cols:    []string{"provider_id"},
				RefTbl:  "ebics_rtn_outbound_providers",
				RefCols: []string{"id"},
			},
			{
				Name:    "ebics_rtn_outbound_notifications_host_id_fkey",
				Cols:    []string{"ebics_host_id"},
				RefTbl:  "ebics_hosts",
				RefCols: []string{"id"},
			},
			{
				Name:    "ebics_rtn_outbound_notifications_subscriber_id_fkey",
				Cols:    []string{"ebics_subscriber_id"},
				RefTbl:  "ebics_subscribers",
				RefCols: []string{"id"},
			},
			{
				Name:    "ebics_rtn_outbound_notifications_reporting_set_id_fkey",
				Cols:    []string{"server_reporting_set_id"},
				RefTbl:  "ebics_server_reporting_sets",
				RefCols: []string{"id"},
			},
		},
	}); err != nil {
		return fmt.Errorf(`failed to create "ebics_rtn_outbound_notifications": %w`, err)
	}

	if err := db.CreateIndex(&Index{
		Name: "ebics_rtn_outbound_notifications_due_idx",
		On:   "ebics_rtn_outbound_notifications",
		Cols: []string{"owner", "status", "next_retry_at", "created_at"},
	}); err != nil {
		return fmt.Errorf(`failed to create index on "ebics_rtn_outbound_notifications": %w`, err)
	}

	return nil
}

func ver0_16_8AddEbicsOutboundRTNTablesDown(db Actions) error {
	if err := db.DropTable("ebics_rtn_outbound_notifications"); err != nil {
		return fmt.Errorf(`failed to drop "ebics_rtn_outbound_notifications": %w`, err)
	}
	if err := db.DropTable("ebics_rtn_outbound_providers"); err != nil {
		return fmt.Errorf(`failed to drop "ebics_rtn_outbound_providers": %w`, err)
	}

	return nil
}
