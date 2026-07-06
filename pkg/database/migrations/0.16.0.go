package migrations

import (
	"fmt"
)

func ver0_16_0AddRemoteFilewatcherTableUp(db Actions) error {
	if err := db.CreateTable("file_watchers", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "owner", Type: Varchar(100), NotNull: true},
			{Name: "disabled", Type: Boolean{}, NotNull: true, Default: false},
			{Name: "flow", Type: Varchar(50), NotNull: true},
			{Name: "interval", Type: BigInt{}, NotNull: true},
			{Name: "pattern", Type: Text{}, NotNull: true},
			{Name: "remote_account_id", Type: BigInt{}, NotNull: true},
			{Name: "client_id", Type: BigInt{}, NotNull: true},
			{Name: "rule_id", Type: BigInt{}, NotNull: true},
			{Name: "no_duplicate_check", Type: Boolean{}, NotNull: true, Default: false},
		},
		PrimaryKey: &PrimaryKey{
			Name: "remote_file_watchers_pkey",
			Cols: []string{"id"},
		},
		ForeignKeys: []ForeignKey{{
			Name: "remote_account_fkey", Cols: []string{"remote_account_id"},
			RefTbl: "remote_accounts", RefCols: []string{"id"},
			OnUpdate: Restrict, OnDelete: Restrict,
		}, {
			Name: "client_fkey", Cols: []string{"client_id"},
			RefTbl: "clients", RefCols: []string{"id"},
			OnUpdate: Restrict, OnDelete: Restrict,
		}, {
			Name: "rule_fkey", Cols: []string{"rule_id"},
			RefTbl: "rules", RefCols: []string{"id"},
			OnUpdate: Restrict, OnDelete: Restrict,
		}},
		Uniques: []Unique{
			{Name: "unique_remotefw_flow", Cols: []string{"owner", "flow"}},
		},
		Checks: nil,
	}); err != nil {
		return fmt.Errorf("failed to create remote filewatchers table: %w", err)
	}

	return nil
}

func ver0_16_0AddRemoteFilewatcherTableDown(db Actions) error {
	if err := db.DropTable("file_watchers"); err != nil {
		return fmt.Errorf("failed to drop remote filewatchers table: %w", err)
	}

	return nil
}

func ver0_16_0AddNormalizedTransferInfoViewUp(db Actions) error {
	if err := db.CreateView(&View{
		Name: "normalized_transfer_info",
		As:   "SELECT COALESCE(transfer_id,history_id) as owner_id, value, name FROM transfer_info",
	}); err != nil {
		return fmt.Errorf("failed to create normalized transfer info view: %w", err)
	}

	return nil
}

func ver0_16_0AddNormalizedTransferInfoViewDown(db Actions) error {
	if err := db.DropView("normalized_transfer_info"); err != nil {
		return fmt.Errorf("failed to drop normalized transfer info view: %w", err)
	}

	return nil
}
