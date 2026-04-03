package migrations

import "fmt"

func ver0_16_4AddEbicsContractRefreshPoliciesUp(db Actions) error {
	if err := db.CreateTable("ebics_contract_refresh_policies", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "owner", Type: Varchar(50), NotNull: true},
			{Name: "name", Type: Varchar(100), NotNull: true},
			{Name: "enabled", Type: Boolean{}, NotNull: true, Default: true},
			{Name: "client_id", Type: BigInt{}, NotNull: true},
			{Name: "ebics_subscriber_id", Type: BigInt{}, NotNull: true},
			{Name: "include_hev", Type: Boolean{}, NotNull: true, Default: true},
			{Name: "interval_seconds", Type: BigInt{}, NotNull: true, Default: 86400},
			{Name: "status", Type: Varchar(20), NotNull: true, Default: "READY"},
			{Name: "next_run_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
			{Name: "last_attempt_at", Type: DateTime{}},
			{Name: "last_success_at", Type: DateTime{}},
			{Name: "last_error", Type: Text{}},
			{Name: "created_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
			{Name: "updated_at", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
		},
		PrimaryKey: &PrimaryKey{Name: "ebics_contract_refresh_policies_pkey", Cols: []string{"id"}},
		Uniques: []Unique{
			{Name: "unique_ebics_contract_refresh_policy_name", Cols: []string{"owner", "name"}},
		},
		ForeignKeys: []ForeignKey{
			{
				Name:    "ebics_contract_refresh_policies_client_id_fkey",
				Cols:    []string{"client_id"},
				RefTbl:  "clients",
				RefCols: []string{"id"},
			},
			{
				Name:    "ebics_contract_refresh_policies_subscriber_id_fkey",
				Cols:    []string{"ebics_subscriber_id"},
				RefTbl:  "ebics_subscribers",
				RefCols: []string{"id"},
			},
		},
	}); err != nil {
		return fmt.Errorf(`failed to create "ebics_contract_refresh_policies": %w`, err)
	}

	return nil
}

func ver0_16_4AddEbicsContractRefreshPoliciesDown(db Actions) error {
	if err := db.DropTable("ebics_contract_refresh_policies"); err != nil {
		return fmt.Errorf(`failed to drop "ebics_contract_refresh_policies": %w`, err)
	}

	return nil
}
