package migrations

import "fmt"

func ver0_16_5AddEbicsHistoryEntriesUp(db Actions) error {
	if err := db.CreateTable("ebics_history_entries", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "owner", Type: Varchar(50), NotNull: true},
			{Name: "history_type", Type: Varchar(20), NotNull: true},
			{Name: "operation_type", Type: Varchar(50), NotNull: true},
			{Name: "action", Type: Varchar(50)},
			{Name: "order_type", Type: Varchar(20)},
			{Name: "direction", Type: Varchar(20)},
			{Name: "transport_mode", Type: Varchar(20)},
			{Name: "status", Type: Varchar(30), NotNull: true},
			{Name: "severity", Type: Varchar(20)},
			{Name: "technical_return_code", Type: Varchar(20)},
			{Name: "technical_return_message", Type: Text{}},
			{Name: "business_return_code", Type: Varchar(20)},
			{Name: "business_return_message", Type: Text{}},
			{Name: "gateway_outcome", Type: Varchar(50)},
			{Name: "retry_decision", Type: Varchar(50)},
			{Name: "client_id", Type: BigInt{}},
			{Name: "ebics_host_id", Type: BigInt{}, NotNull: true},
			{Name: "ebics_subscriber_id", Type: BigInt{}, NotNull: true},
			{Name: "operation_id", Type: BigInt{}},
			{Name: "transfer_id", Type: BigInt{}},
			{Name: "workflow_id", Type: BigInt{}},
			{Name: "lifecycle_id", Type: BigInt{}},
			{Name: "coordination_id", Type: Varchar(100)},
			{Name: "request_id", Type: Varchar(100)},
			{Name: "correlation_id", Type: Varchar(100)},
			{Name: "transaction_id", Type: Varchar(100)},
			{Name: "operator", Type: Varchar(100)},
			{Name: "reason", Type: Text{}},
			{Name: "evidence", Type: Text{}, NotNull: true, Default: "{}"},
			{Name: "metadata", Type: Text{}, NotNull: true, Default: "{}"},
			{Name: "started_at", Type: DateTime{}},
			{Name: "finished_at", Type: DateTime{}},
			{Name: "created_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
		},
		PrimaryKey: &PrimaryKey{Name: "ebics_history_entries_pkey", Cols: []string{"id"}},
		ForeignKeys: []ForeignKey{
			{
				Name:    "ebics_history_entries_client_id_fkey",
				Cols:    []string{"client_id"},
				RefTbl:  "clients",
				RefCols: []string{"id"},
			},
			{
				Name:    "ebics_history_entries_host_id_fkey",
				Cols:    []string{"ebics_host_id"},
				RefTbl:  "ebics_hosts",
				RefCols: []string{"id"},
			},
			{
				Name:    "ebics_history_entries_subscriber_id_fkey",
				Cols:    []string{"ebics_subscriber_id"},
				RefTbl:  "ebics_subscribers",
				RefCols: []string{"id"},
			},
			{
				Name:    "ebics_history_entries_operation_id_fkey",
				Cols:    []string{"operation_id"},
				RefTbl:  "ebics_operations",
				RefCols: []string{"id"},
			},
			{
				Name:    "ebics_history_entries_transfer_id_fkey",
				Cols:    []string{"transfer_id"},
				RefTbl:  "transfers",
				RefCols: []string{"id"},
			},
			{
				Name:    "ebics_history_entries_workflow_id_fkey",
				Cols:    []string{"workflow_id"},
				RefTbl:  "ebics_initialization_workflows",
				RefCols: []string{"id"},
			},
			{
				Name:    "ebics_history_entries_lifecycle_id_fkey",
				Cols:    []string{"lifecycle_id"},
				RefTbl:  "ebics_key_lifecycles",
				RefCols: []string{"id"},
			},
		},
	}); err != nil {
		return fmt.Errorf(`failed to create "ebics_history_entries": %w`, err)
	}

	for _, index := range []*Index{
		{Name: "ebics_history_entries_created_at_idx", On: "ebics_history_entries", Cols: []string{"created_at"}},
		{
			Name: "ebics_history_entries_subscriber_idx",
			On:   "ebics_history_entries",
			Cols: []string{"ebics_subscriber_id", "created_at"},
		},
		{Name: "ebics_history_entries_operation_idx", On: "ebics_history_entries", Cols: []string{"operation_id"}},
		{Name: "ebics_history_entries_workflow_idx", On: "ebics_history_entries", Cols: []string{"workflow_id"}},
		{Name: "ebics_history_entries_lifecycle_idx", On: "ebics_history_entries", Cols: []string{"lifecycle_id"}},
		{Name: "ebics_history_entries_coordination_idx", On: "ebics_history_entries", Cols: []string{"coordination_id"}},
	} {
		if err := db.CreateIndex(index); err != nil {
			return fmt.Errorf(`failed to create index %q on "ebics_history_entries": %w`, index.Name, err)
		}
	}

	return nil
}

func ver0_16_5AddEbicsHistoryEntriesDown(db Actions) error {
	if err := db.DropTable("ebics_history_entries"); err != nil {
		return fmt.Errorf(`failed to drop "ebics_history_entries": %w`, err)
	}

	return nil
}
