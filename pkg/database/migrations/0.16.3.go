package migrations

import "fmt"

func ver0_16_3AddEbicsRuntimePoliciesUp(db Actions) error {
	if err := db.CreateTable("ebics_runtime_policies", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "owner", Type: Varchar(50), NotNull: true},
			{Name: "name", Type: Varchar(100), NotNull: true},
			{Name: "enabled", Type: Boolean{}, NotNull: true, Default: true},
			{Name: "maintenance_interval_seconds", Type: BigInt{}, NotNull: true, Default: 21600},
			{Name: "transaction_retention_seconds", Type: BigInt{}, NotNull: true, Default: 604800},
			{Name: "rtn_event_retention_seconds", Type: BigInt{}, NotNull: true, Default: 2592000},
			{Name: "created_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
			{Name: "updated_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
		},
		PrimaryKey: &PrimaryKey{Name: "ebics_runtime_policies_pkey", Cols: []string{"id"}},
		Uniques: []Unique{
			{Name: "unique_ebics_runtime_policy_name", Cols: []string{"owner", "name"}},
		},
	}); err != nil {
		return fmt.Errorf(`failed to create "ebics_runtime_policies": %w`, err)
	}

	return nil
}

func ver0_16_3AddEbicsRuntimePoliciesDown(db Actions) error {
	if err := db.DropTable("ebics_runtime_policies"); err != nil {
		return fmt.Errorf(`failed to drop "ebics_runtime_policies": %w`, err)
	}

	return nil
}
