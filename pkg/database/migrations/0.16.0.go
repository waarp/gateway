package migrations

import "fmt"

func ver0_16_0AddEbicsTablesUp(db Actions) error {
	steps := []func(Actions) error{
		ver0_16_0CreateEbicsHosts,
		ver0_16_0CreateEbicsSubscribers,
		ver0_16_0CreateEbicsSubscriberKeyMaterials,
		ver0_16_0CreateEbicsBankKeys,
		ver0_16_0CreateEbicsPayloadProfiles,
		ver0_16_0CreateEbicsOperations,
		ver0_16_0CreateEbicsContractViews,
		ver0_16_0CreateEbicsContractViewItems,
		ver0_16_0CreateEbicsRTNEvents,
		ver0_16_0CreateEbicsTransactions,
		ver0_16_0CreateEbicsTransactionSegments,
		ver0_16_0CreateEbicsNonces,
		ver0_16_0CreateEbicsKeyLifecycles,
		ver0_16_0CreateEbicsInitializationWorkflows,
		ver0_16_0CreateEbicsRTNProviders,
	}

	for _, step := range steps {
		if err := step(db); err != nil {
			return err
		}
	}

	if err := db.AlterTable("ebics_operations",
		AddForeignKey{
			Name:   "ebics_operations_contract_view_fkey",
			Cols:   []string{"contract_view_id"},
			RefTbl: "ebics_contract_views", RefCols: []string{"id"},
			OnUpdate: Restrict, OnDelete: SetNull,
		},
		AddForeignKey{
			Name:   "ebics_operations_rtn_event_fkey",
			Cols:   []string{"rtn_event_id"},
			RefTbl: "ebics_rtn_events", RefCols: []string{"id"},
			OnUpdate: Restrict, OnDelete: SetNull,
		},
	); err != nil {
		return fmt.Errorf("failed to alter ebics_operations foreign keys: %w", err)
	}

	return nil
}

func ver0_16_0AddEbicsTablesDown(db Actions) error {
	for _, table := range []string{
		"ebics_rtn_providers",
		"ebics_initialization_workflows",
		"ebics_key_lifecycles",
		"ebics_nonces",
		"ebics_transaction_segments",
		"ebics_transactions",
		"ebics_rtn_events",
		"ebics_contract_view_items",
		"ebics_contract_views",
		"ebics_operations",
		"ebics_payload_profiles",
		"ebics_bank_keys",
		"ebics_subscriber_key_materials",
		"ebics_subscribers",
		"ebics_hosts",
	} {
		if err := db.DropTable(table); err != nil {
			return fmt.Errorf("failed to drop %q: %w", table, err)
		}
	}

	return nil
}

func ver0_16_0CreateEbicsHosts(db Actions) error {
	err := db.CreateTable("ebics_hosts", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "owner", Type: Varchar(50), NotNull: true},
			{Name: "name", Type: Varchar(100), NotNull: true},
			{Name: "host_id", Type: Varchar(100), NotNull: true},
			{Name: "description", Type: Text{}, NotNull: true, Default: ""},
			{Name: "enabled", Type: Boolean{}, NotNull: true, Default: true},
			{Name: "is_server", Type: Boolean{}, NotNull: true, Default: false},
			{Name: "protocol_version", Type: Varchar(10), NotNull: true},
			{Name: "transport", Type: Varchar(20), NotNull: true},
			{Name: "default_bank_url", Type: Text{}, NotNull: true, Default: ""},
			{Name: "created_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
			{Name: "updated_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
		},
		PrimaryKey: &PrimaryKey{Name: "ebics_hosts_pkey", Cols: []string{"id"}},
		Uniques: []Unique{
			{Name: "unique_ebics_host_name", Cols: []string{"owner", "name"}},
			{Name: "unique_ebics_host_id", Cols: []string{"owner", "host_id"}},
		},
	})
	if err != nil {
		return fmt.Errorf(`failed to create "ebics_hosts": %w`, err)
	}

	return nil
}

func ver0_16_0CreateEbicsSubscribers(db Actions) error {
	err := db.CreateTable("ebics_subscribers", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "owner", Type: Varchar(50), NotNull: true},
			{Name: "ebics_host_id", Type: BigInt{}, NotNull: true},
			{Name: "name", Type: Varchar(100), NotNull: true},
			{Name: "partner_id", Type: Varchar(100), NotNull: true},
			{Name: "user_id", Type: Varchar(100), NotNull: true},
			{Name: "system_id", Type: Varchar(100), NotNull: true, Default: ""},
			{Name: "local_account_id", Type: BigInt{}},
			{Name: "remote_account_id", Type: BigInt{}},
			{Name: "account_role", Type: Varchar(20), NotNull: true, Default: ""},
			{Name: "transport_url", Type: Text{}, NotNull: true, Default: ""},
			{Name: "enabled", Type: Boolean{}, NotNull: true, Default: true},
			{Name: "default_order_data_encoding", Type: Varchar(50), NotNull: true, Default: ""},
			{Name: "created_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
			{Name: "updated_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
		},
		PrimaryKey: &PrimaryKey{Name: "ebics_subscribers_pkey", Cols: []string{"id"}},
		ForeignKeys: []ForeignKey{{
			Name: "ebics_subscribers_host_fkey", Cols: []string{"ebics_host_id"},
			RefTbl: "ebics_hosts", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: Cascade,
		}, {
			Name: "ebics_subscribers_local_account_fkey", Cols: []string{"local_account_id"},
			RefTbl: "local_accounts", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: Restrict,
		}, {
			Name: "ebics_subscribers_remote_account_fkey", Cols: []string{"remote_account_id"},
			RefTbl: "remote_accounts", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: Restrict,
		}},
		Uniques: []Unique{
			{Name: "unique_ebics_subscriber_name", Cols: []string{"owner", "name"}},
			{Name: "unique_ebics_subscriber_identity", Cols: []string{"owner", "ebics_host_id", "partner_id", "user_id"}},
		},
	})
	if err != nil {
		return fmt.Errorf(`failed to create "ebics_subscribers": %w`, err)
	}

	return nil
}

func ver0_16_0CreateEbicsSubscriberKeyMaterials(db Actions) error {
	err := db.CreateTable("ebics_subscriber_key_materials", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "owner", Type: Varchar(50), NotNull: true},
			{Name: "ebics_subscriber_id", Type: BigInt{}, NotNull: true},
			{Name: "key_usage", Type: Varchar(30), NotNull: true},
			{Name: "public_key", Type: Text{}, NotNull: true, Default: ""},
			{Name: "public_key_version", Type: Varchar(20), NotNull: true, Default: ""},
			{Name: "certificate", Type: Text{}, NotNull: true, Default: ""},
			{Name: "certificate_version", Type: Varchar(20), NotNull: true, Default: ""},
			{Name: "state", Type: Varchar(20), NotNull: true},
			{Name: "created_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
			{Name: "updated_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
		},
		PrimaryKey: &PrimaryKey{Name: "ebics_subscriber_key_materials_pkey", Cols: []string{"id"}},
		ForeignKeys: []ForeignKey{{
			Name: "ebics_subscriber_key_materials_subscriber_fkey", Cols: []string{"ebics_subscriber_id"},
			RefTbl: "ebics_subscribers", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: Cascade,
		}},
	})
	if err != nil {
		return fmt.Errorf(`failed to create "ebics_subscriber_key_materials": %w`, err)
	}

	return nil
}

func ver0_16_0CreateEbicsBankKeys(db Actions) error {
	err := db.CreateTable("ebics_bank_keys", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "owner", Type: Varchar(50), NotNull: true},
			{Name: "ebics_host_id", Type: BigInt{}, NotNull: true},
			{Name: "key_type", Type: Varchar(20), NotNull: true},
			{Name: "version", Type: Varchar(20), NotNull: true, Default: ""},
			{Name: "public_key", Type: Text{}, NotNull: true},
			{Name: "public_key_hash", Type: Text{}, NotNull: true, Default: ""},
			{Name: "state", Type: Varchar(20), NotNull: true},
			{Name: "valid_from", Type: DateTime{}},
			{Name: "valid_to", Type: DateTime{}},
			{Name: "created_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
			{Name: "updated_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
		},
		PrimaryKey: &PrimaryKey{Name: "ebics_bank_keys_pkey", Cols: []string{"id"}},
		ForeignKeys: []ForeignKey{{
			Name: "ebics_bank_keys_host_fkey", Cols: []string{"ebics_host_id"},
			RefTbl: "ebics_hosts", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: Cascade,
		}},
		Uniques: []Unique{{
			Name: "unique_ebics_bank_key_identity",
			Cols: []string{"owner", "ebics_host_id", "key_type", "version"},
		}},
	})
	if err != nil {
		return fmt.Errorf(`failed to create "ebics_bank_keys": %w`, err)
	}

	return nil
}

func ver0_16_0CreateEbicsPayloadProfiles(db Actions) error {
	err := db.CreateTable("ebics_payload_profiles", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "owner", Type: Varchar(50), NotNull: true},
			{Name: "name", Type: Varchar(100), NotNull: true},
			{Name: "label", Type: Varchar(255), NotNull: true, Default: ""},
			{Name: "description", Type: Text{}, NotNull: true, Default: ""},
			{Name: "order_type", Type: Varchar(20), NotNull: true},
			{Name: "direction", Type: Varchar(20), NotNull: true},
			{Name: "service_name", Type: Varchar(50), NotNull: true, Default: ""},
			{Name: "service_option", Type: Varchar(50), NotNull: true, Default: ""},
			{Name: "scope", Type: Varchar(20), NotNull: true, Default: ""},
			{Name: "msg_name", Type: Varchar(100), NotNull: true, Default: ""},
			{Name: "container_type", Type: Varchar(100), NotNull: true, Default: ""},
			{Name: "default_rule_id", Type: BigInt{}},
			{Name: "default_target_directory", Type: Text{}, NotNull: true, Default: ""},
			{Name: "requires_declared_amount", Type: Boolean{}, NotNull: true, Default: false},
			{Name: "default_currency", Type: Varchar(10), NotNull: true, Default: ""},
			{Name: "allowed_extensions", Type: Text{}, NotNull: true, Default: "[]"},
			{Name: "filename_pattern", Type: Varchar(255), NotNull: true, Default: ""},
			{Name: "strict_contract_check", Type: Boolean{}, NotNull: true, Default: false},
			{Name: "is_enabled", Type: Boolean{}, NotNull: true, Default: true},
			{Name: "metadata", Type: Text{}, NotNull: true, Default: "{}"},
			{Name: "created_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
			{Name: "updated_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
		},
		PrimaryKey: &PrimaryKey{Name: "ebics_payload_profiles_pkey", Cols: []string{"id"}},
		ForeignKeys: []ForeignKey{{
			Name: "ebics_payload_profiles_rule_fkey", Cols: []string{"default_rule_id"},
			RefTbl: "rules", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: SetNull,
		}},
		Uniques: []Unique{{Name: "unique_ebics_payload_profile_name", Cols: []string{"owner", "name"}}},
	})
	if err != nil {
		return fmt.Errorf(`failed to create "ebics_payload_profiles": %w`, err)
	}

	return nil
}

func ver0_16_0CreateEbicsOperations(db Actions) error {
	err := db.CreateTable("ebics_operations", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "owner", Type: Varchar(50), NotNull: true},
			{Name: "local_agent_id", Type: BigInt{}},
			{Name: "client_id", Type: BigInt{}},
			{Name: "remote_agent_id", Type: BigInt{}},
			{Name: "local_account_id", Type: BigInt{}},
			{Name: "remote_account_id", Type: BigInt{}},
			{Name: "ebics_host_id", Type: BigInt{}, NotNull: true},
			{Name: "ebics_subscriber_id", Type: BigInt{}, NotNull: true},
			{Name: "operation_type", Type: Varchar(40), NotNull: true},
			{Name: "order_type", Type: Varchar(20), NotNull: true},
			{Name: "direction", Type: Varchar(20), NotNull: true},
			{Name: "transport_mode", Type: Varchar(20), NotNull: true},
			{Name: "transaction_id", Type: Varchar(100), NotNull: true, Default: ""},
			{Name: "request_id", Type: Varchar(100), NotNull: true, Default: ""},
			{Name: "correlation_id", Type: Varchar(100), NotNull: true, Default: ""},
			{Name: "ebics_version", Type: Varchar(10), NotNull: true, Default: ""},
			{Name: "status", Type: Varchar(50), NotNull: true},
			{Name: "severity", Type: Varchar(20), NotNull: true},
			{Name: "technical_return_code", Type: Varchar(100), NotNull: true, Default: ""},
			{Name: "technical_return_message", Type: Text{}, NotNull: true, Default: ""},
			{Name: "business_return_code", Type: Varchar(100), NotNull: true, Default: ""},
			{Name: "business_return_message", Type: Text{}, NotNull: true, Default: ""},
			{Name: "gateway_outcome", Type: Varchar(60), NotNull: true},
			{Name: "retry_decision", Type: Varchar(60), NotNull: true},
			{Name: "manual_action_required", Type: Boolean{}, NotNull: true, Default: false},
			{Name: "transfer_id", Type: BigInt{}},
			{Name: "contract_view_id", Type: BigInt{}},
			{Name: "rtn_event_id", Type: BigInt{}},
			{Name: "metadata", Type: Text{}, NotNull: true, Default: "{}"},
			{Name: "started_at", Type: DateTime{}},
			{Name: "finished_at", Type: DateTime{}},
			{Name: "created_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
			{Name: "updated_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
		},
		PrimaryKey: &PrimaryKey{Name: "ebics_operations_pkey", Cols: []string{"id"}},
		ForeignKeys: []ForeignKey{{
			Name: "ebics_operations_local_agent_fkey", Cols: []string{"local_agent_id"},
			RefTbl: "local_agents", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: SetNull,
		}, {
			Name: "ebics_operations_client_fkey", Cols: []string{"client_id"},
			RefTbl: "clients", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: SetNull,
		}, {
			Name: "ebics_operations_remote_agent_fkey", Cols: []string{"remote_agent_id"},
			RefTbl: "remote_agents", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: SetNull,
		}, {
			Name: "ebics_operations_local_account_fkey", Cols: []string{"local_account_id"},
			RefTbl: "local_accounts", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: SetNull,
		}, {
			Name: "ebics_operations_remote_account_fkey", Cols: []string{"remote_account_id"},
			RefTbl: "remote_accounts", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: SetNull,
		}, {
			Name: "ebics_operations_host_fkey", Cols: []string{"ebics_host_id"},
			RefTbl: "ebics_hosts", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: Restrict,
		}, {
			Name: "ebics_operations_subscriber_fkey", Cols: []string{"ebics_subscriber_id"},
			RefTbl: "ebics_subscribers", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: Restrict,
		}, {
			Name: "ebics_operations_transfer_fkey", Cols: []string{"transfer_id"},
			RefTbl: "transfers", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: SetNull,
		}},
	})
	if err != nil {
		return fmt.Errorf(`failed to create "ebics_operations": %w`, err)
	}

	return nil
}

func ver0_16_0CreateEbicsContractViews(db Actions) error {
	err := db.CreateTable("ebics_contract_views", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "owner", Type: Varchar(50), NotNull: true},
			{Name: "ebics_host_id", Type: BigInt{}, NotNull: true},
			{Name: "ebics_subscriber_id", Type: BigInt{}},
			{Name: "source_order_type", Type: Varchar(20), NotNull: true},
			{Name: "source_operation_id", Type: BigInt{}},
			{Name: "version_tag", Type: Varchar(100), NotNull: true, Default: ""},
			{Name: "status", Type: Varchar(30), NotNull: true},
			{Name: "fetched_at", Type: DateTime{}, NotNull: true},
			{Name: "created_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
			{Name: "updated_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
		},
		PrimaryKey: &PrimaryKey{Name: "ebics_contract_views_pkey", Cols: []string{"id"}},
		ForeignKeys: []ForeignKey{{
			Name: "ebics_contract_views_host_fkey", Cols: []string{"ebics_host_id"},
			RefTbl: "ebics_hosts", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: Restrict,
		}, {
			Name: "ebics_contract_views_subscriber_fkey", Cols: []string{"ebics_subscriber_id"},
			RefTbl: "ebics_subscribers", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: SetNull,
		}, {
			Name: "ebics_contract_views_operation_fkey", Cols: []string{"source_operation_id"},
			RefTbl: "ebics_operations", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: SetNull,
		}},
	})
	if err != nil {
		return fmt.Errorf(`failed to create "ebics_contract_views": %w`, err)
	}

	return nil
}

func ver0_16_0CreateEbicsContractViewItems(db Actions) error {
	err := db.CreateTable("ebics_contract_view_items", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "owner", Type: Varchar(50), NotNull: true},
			{Name: "contract_view_id", Type: BigInt{}, NotNull: true},
			{Name: "item_type", Type: Varchar(30), NotNull: true},
			{Name: "item_key", Type: Varchar(255), NotNull: true},
			{Name: "order_type", Type: Varchar(20), NotNull: true, Default: ""},
			{Name: "service_name", Type: Varchar(50), NotNull: true, Default: ""},
			{Name: "service_option", Type: Varchar(50), NotNull: true, Default: ""},
			{Name: "scope", Type: Varchar(20), NotNull: true, Default: ""},
			{Name: "msg_name", Type: Varchar(100), NotNull: true, Default: ""},
			{Name: "container_type", Type: Varchar(100), NotNull: true, Default: ""},
			{Name: "admin_order_type", Type: Varchar(20), NotNull: true, Default: ""},
			{Name: "authorisation_level", Type: Varchar(50), NotNull: true, Default: ""},
			{Name: "account_id", Type: Varchar(100), NotNull: true, Default: ""},
			{Name: "max_amount_value", Type: Varchar(50), NotNull: true, Default: ""},
			{Name: "max_amount_currency", Type: Varchar(10), NotNull: true, Default: ""},
			{Name: "is_enabled", Type: Boolean{}, NotNull: true, Default: true},
			{Name: "payload", Type: Text{}, NotNull: true, Default: ""},
			{Name: "created_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
			{Name: "updated_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
		},
		PrimaryKey: &PrimaryKey{Name: "ebics_contract_view_items_pkey", Cols: []string{"id"}},
		ForeignKeys: []ForeignKey{{
			Name: "ebics_contract_view_items_view_fkey", Cols: []string{"contract_view_id"},
			RefTbl: "ebics_contract_views", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: Cascade,
		}},
	})
	if err != nil {
		return fmt.Errorf(`failed to create "ebics_contract_view_items": %w`, err)
	}

	return nil
}

func ver0_16_0CreateEbicsRTNEvents(db Actions) error {
	err := db.CreateTable("ebics_rtn_events", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "owner", Type: Varchar(50), NotNull: true},
			{Name: "source", Type: Varchar(100), NotNull: true},
			{Name: "event_id", Type: Varchar(255), NotNull: true, Default: ""},
			{Name: "correlation_id", Type: Varchar(255), NotNull: true, Default: ""},
			{Name: "idempotence_key", Type: Varchar(255), NotNull: true},
			{Name: "ebics_host_id", Type: BigInt{}},
			{Name: "ebics_subscriber_id", Type: BigInt{}},
			{Name: "order_type_hint", Type: Varchar(20), NotNull: true, Default: ""},
			{Name: "profile_id", Type: Varchar(100), NotNull: true, Default: ""},
			{Name: "payload", Type: Text{}, NotNull: true, Default: "{}"},
			{Name: "status", Type: Varchar(30), NotNull: true},
			{Name: "attempts", Type: Integer{}, NotNull: true, Default: 0},
			{Name: "next_retry_at", Type: DateTime{}},
			{Name: "received_at", Type: DateTime{}, NotNull: true},
			{Name: "processed_at", Type: DateTime{}},
			{Name: "last_error", Type: Text{}, NotNull: true, Default: ""},
			{Name: "created_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
			{Name: "updated_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
		},
		PrimaryKey: &PrimaryKey{Name: "ebics_rtn_events_pkey", Cols: []string{"id"}},
		ForeignKeys: []ForeignKey{{
			Name: "ebics_rtn_events_host_fkey", Cols: []string{"ebics_host_id"},
			RefTbl: "ebics_hosts", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: SetNull,
		}, {
			Name: "ebics_rtn_events_subscriber_fkey", Cols: []string{"ebics_subscriber_id"},
			RefTbl: "ebics_subscribers", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: SetNull,
		}},
		Uniques: []Unique{{Name: "unique_ebics_rtn_idempotence_key", Cols: []string{"owner", "idempotence_key"}}},
	})
	if err != nil {
		return fmt.Errorf(`failed to create "ebics_rtn_events": %w`, err)
	}

	return nil
}

func ver0_16_0CreateEbicsTransactions(db Actions) error {
	err := db.CreateTable("ebics_transactions", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "owner", Type: Varchar(50), NotNull: true},
			{Name: "ebics_operation_id", Type: BigInt{}},
			{Name: "ebics_host_id", Type: BigInt{}, NotNull: true},
			{Name: "ebics_subscriber_id", Type: BigInt{}, NotNull: true},
			{Name: "transaction_id", Type: Varchar(100), NotNull: true},
			{Name: "order_type", Type: Varchar(20), NotNull: true},
			{Name: "transfer_id", Type: BigInt{}},
			{Name: "status", Type: Varchar(30), NotNull: true},
			{Name: "direction", Type: Varchar(20), NotNull: true},
			{Name: "segment_count", Type: Integer{}, NotNull: true, Default: 0},
			{Name: "current_segment", Type: Integer{}, NotNull: true, Default: 0},
			{Name: "total_size", Type: BigInt{}, NotNull: true, Default: 0},
			{Name: "resumed_from_tx_id", Type: BigInt{}},
			{Name: "metadata", Type: Text{}, NotNull: true, Default: "{}"},
			{Name: "created_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
			{Name: "updated_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
		},
		PrimaryKey: &PrimaryKey{Name: "ebics_transactions_pkey", Cols: []string{"id"}},
		ForeignKeys: []ForeignKey{{
			Name: "ebics_transactions_operation_fkey", Cols: []string{"ebics_operation_id"},
			RefTbl: "ebics_operations", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: SetNull,
		}, {
			Name: "ebics_transactions_host_fkey", Cols: []string{"ebics_host_id"},
			RefTbl: "ebics_hosts", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: Restrict,
		}, {
			Name: "ebics_transactions_subscriber_fkey", Cols: []string{"ebics_subscriber_id"},
			RefTbl: "ebics_subscribers", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: Restrict,
		}, {
			Name: "ebics_transactions_transfer_fkey", Cols: []string{"transfer_id"},
			RefTbl: "transfers", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: SetNull,
		}, {
			Name: "ebics_transactions_resumed_from_fkey", Cols: []string{"resumed_from_tx_id"},
			RefTbl: "ebics_transactions", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: SetNull,
		}},
		Uniques: []Unique{{
			Name: "unique_ebics_transaction_identity",
			Cols: []string{"owner", "ebics_subscriber_id", "transaction_id"},
		}},
	})
	if err != nil {
		return fmt.Errorf(`failed to create "ebics_transactions": %w`, err)
	}

	return nil
}

func ver0_16_0CreateEbicsTransactionSegments(db Actions) error {
	err := db.CreateTable("ebics_transaction_segments", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "owner", Type: Varchar(50), NotNull: true},
			{Name: "ebics_transaction_id", Type: BigInt{}, NotNull: true},
			{Name: "segment_number", Type: Integer{}, NotNull: true},
			{Name: "segment_status", Type: Varchar(30), NotNull: true},
			{Name: "payload_size", Type: BigInt{}, NotNull: true, Default: 0},
			{Name: "checksum", Type: Varchar(255), NotNull: true, Default: ""},
			{Name: "stored_payload_ref", Type: Text{}, NotNull: true, Default: ""},
			{Name: "metadata", Type: Text{}, NotNull: true, Default: "{}"},
			{Name: "created_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
			{Name: "updated_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
		},
		PrimaryKey: &PrimaryKey{Name: "ebics_transaction_segments_pkey", Cols: []string{"id"}},
		ForeignKeys: []ForeignKey{{
			Name: "ebics_transaction_segments_tx_fkey", Cols: []string{"ebics_transaction_id"},
			RefTbl: "ebics_transactions", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: Cascade,
		}},
		Uniques: []Unique{{
			Name: "unique_ebics_transaction_segment",
			Cols: []string{"owner", "ebics_transaction_id", "segment_number"},
		}},
	})
	if err != nil {
		return fmt.Errorf(`failed to create "ebics_transaction_segments": %w`, err)
	}

	return nil
}

func ver0_16_0CreateEbicsNonces(db Actions) error {
	err := db.CreateTable("ebics_nonces", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "owner", Type: Varchar(50), NotNull: true},
			{Name: "ebics_subscriber_id", Type: BigInt{}, NotNull: true},
			{Name: "nonce", Type: Varchar(255), NotNull: true},
			{Name: "timestamp", Type: DateTime{}, NotNull: true},
			{Name: "expires_at", Type: DateTime{}, NotNull: true},
			{Name: "created_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
			{Name: "updated_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
		},
		PrimaryKey: &PrimaryKey{Name: "ebics_nonces_pkey", Cols: []string{"id"}},
		ForeignKeys: []ForeignKey{{
			Name: "ebics_nonces_subscriber_fkey", Cols: []string{"ebics_subscriber_id"},
			RefTbl: "ebics_subscribers", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: Cascade,
		}},
		Uniques: []Unique{{Name: "unique_ebics_nonce_value", Cols: []string{"owner", "ebics_subscriber_id", "nonce"}}},
	})
	if err != nil {
		return fmt.Errorf(`failed to create "ebics_nonces": %w`, err)
	}

	return nil
}

func ver0_16_0CreateEbicsKeyLifecycles(db Actions) error {
	err := db.CreateTable("ebics_key_lifecycles", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "owner", Type: Varchar(50), NotNull: true},
			{Name: "ebics_subscriber_id", Type: BigInt{}, NotNull: true},
			{Name: "key_usage", Type: Varchar(30), NotNull: true},
			{Name: "rotation_type", Type: Varchar(30), NotNull: true},
			{Name: "coordination_id", Type: Varchar(255), NotNull: true, Default: ""},
			{Name: "status", Type: Varchar(50), NotNull: true},
			{Name: "current_credential_id", Type: BigInt{}, NotNull: true},
			{Name: "next_credential_id", Type: BigInt{}},
			{Name: "trigger_operation_id", Type: BigInt{}},
			{Name: "last_operation_id", Type: BigInt{}},
			{Name: "requested_at", Type: DateTime{}},
			{Name: "sent_at", Type: DateTime{}},
			{Name: "activated_at", Type: DateTime{}},
			{Name: "retired_at", Type: DateTime{}},
			{Name: "operator", Type: Varchar(255), NotNull: true, Default: ""},
			{Name: "reason", Type: Text{}, NotNull: true, Default: ""},
			{Name: "evidence", Type: Text{}, NotNull: true, Default: "{}"},
			{Name: "created_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
			{Name: "updated_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
		},
		PrimaryKey: &PrimaryKey{Name: "ebics_key_lifecycles_pkey", Cols: []string{"id"}},
		ForeignKeys: []ForeignKey{{
			Name: "ebics_key_lifecycles_subscriber_fkey", Cols: []string{"ebics_subscriber_id"},
			RefTbl: "ebics_subscribers", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: Cascade,
		}, {
			Name: "ebics_key_lifecycles_current_credential_fkey", Cols: []string{"current_credential_id"},
			RefTbl: "credentials", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: Restrict,
		}, {
			Name: "ebics_key_lifecycles_next_credential_fkey", Cols: []string{"next_credential_id"},
			RefTbl: "credentials", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: Restrict,
		}, {
			Name: "ebics_key_lifecycles_trigger_operation_fkey", Cols: []string{"trigger_operation_id"},
			RefTbl: "ebics_operations", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: SetNull,
		}, {
			Name: "ebics_key_lifecycles_last_operation_fkey", Cols: []string{"last_operation_id"},
			RefTbl: "ebics_operations", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: SetNull,
		}},
	})
	if err != nil {
		return fmt.Errorf(`failed to create "ebics_key_lifecycles": %w`, err)
	}

	return nil
}

func ver0_16_0CreateEbicsInitializationWorkflows(db Actions) error {
	err := db.CreateTable("ebics_initialization_workflows", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "owner", Type: Varchar(50), NotNull: true},
			{Name: "ebics_subscriber_id", Type: BigInt{}, NotNull: true},
			{Name: "status", Type: Varchar(50), NotNull: true},
			{Name: "current_step", Type: Varchar(50), NotNull: true},
			{Name: "ini_operation_id", Type: BigInt{}},
			{Name: "hia_operation_id", Type: BigInt{}},
			{Name: "h3k_operation_id", Type: BigInt{}},
			{Name: "letter_generated_at", Type: DateTime{}},
			{Name: "letter_confirmed_at", Type: DateTime{}},
			{Name: "bank_activation_at", Type: DateTime{}},
			{Name: "operator", Type: Varchar(255), NotNull: true, Default: ""},
			{Name: "reason", Type: Text{}, NotNull: true, Default: ""},
			{Name: "bank_feedback", Type: Text{}, NotNull: true, Default: ""},
			{Name: "evidence", Type: Text{}, NotNull: true, Default: "{}"},
			{Name: "created_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
			{Name: "updated_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
		},
		PrimaryKey: &PrimaryKey{Name: "ebics_initialization_workflows_pkey", Cols: []string{"id"}},
		ForeignKeys: []ForeignKey{{
			Name: "ebics_initialization_workflows_subscriber_fkey", Cols: []string{"ebics_subscriber_id"},
			RefTbl: "ebics_subscribers", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: Cascade,
		}, {
			Name: "ebics_initialization_workflows_ini_operation_fkey", Cols: []string{"ini_operation_id"},
			RefTbl: "ebics_operations", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: SetNull,
		}, {
			Name: "ebics_initialization_workflows_hia_operation_fkey", Cols: []string{"hia_operation_id"},
			RefTbl: "ebics_operations", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: SetNull,
		}, {
			Name: "ebics_initialization_workflows_h3k_operation_fkey", Cols: []string{"h3k_operation_id"},
			RefTbl: "ebics_operations", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: SetNull,
		}},
	})
	if err != nil {
		return fmt.Errorf(`failed to create "ebics_initialization_workflows": %w`, err)
	}

	return nil
}

func ver0_16_0CreateEbicsRTNProviders(db Actions) error {
	err := db.CreateTable("ebics_rtn_providers", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "owner", Type: Varchar(50), NotNull: true},
			{Name: "name", Type: Varchar(100), NotNull: true},
			{Name: "transport", Type: Varchar(20), NotNull: true},
			{Name: "enabled", Type: Boolean{}, NotNull: true, Default: true},
			{Name: "ebics_subscriber_id", Type: BigInt{}, NotNull: true},
			{Name: "configuration", Type: Text{}, NotNull: true, Default: "{}"},
			{Name: "auto_pull_policy", Type: Varchar(30), NotNull: true},
			{Name: "last_connection_at", Type: DateTime{}},
			{Name: "last_error", Type: Text{}, NotNull: true, Default: ""},
			{Name: "created_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
			{Name: "updated_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
		},
		PrimaryKey: &PrimaryKey{Name: "ebics_rtn_providers_pkey", Cols: []string{"id"}},
		ForeignKeys: []ForeignKey{{
			Name: "ebics_rtn_providers_subscriber_fkey", Cols: []string{"ebics_subscriber_id"},
			RefTbl: "ebics_subscribers", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: Cascade,
		}},
		Uniques: []Unique{{Name: "unique_ebics_rtn_provider_name", Cols: []string{"owner", "name"}}},
	})
	if err != nil {
		return fmt.Errorf(`failed to create "ebics_rtn_providers": %w`, err)
	}

	return nil
}
