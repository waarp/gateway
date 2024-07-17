package migrations

import (
	"fmt"
)

func ver0_10_0AddSNMPMonitorsUp(db Actions) error {
	if err := db.CreateTable("snmp_monitors", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "name", Type: Text{}, NotNull: true},
			{Name: "owner", Type: Text{}, NotNull: true},
			{Name: "udp_address", Type: Text{}, NotNull: true},
			{Name: "snmp_version", Type: Text{}, NotNull: true},
			{Name: "community", Type: Text{}, NotNull: true, Default: "public"},
			{Name: "use_informs", Type: Boolean{}, NotNull: true, Default: false},
			{Name: "snmp_v3_security", Type: Text{}, NotNull: true, Default: false},
			{Name: "snmp_v3_context_name", Type: Text{}, NotNull: true, Default: ""},
			{Name: "snmp_v3_context_engine_id", Type: Text{}, NotNull: true, Default: ""},
			{Name: "snmp_v3_auth_engine_id", Type: Text{}, NotNull: true, Default: ""},
			{Name: "snmp_v3_auth_username", Type: Text{}, NotNull: true, Default: ""},
			{Name: "snmp_v3_auth_protocol", Type: Text{}, NotNull: true, Default: ""},
			{Name: "snmp_v3_auth_passphrase", Type: Text{}, NotNull: true, Default: ""},
			{Name: "snmp_v3_priv_protocol", Type: Text{}, NotNull: true, Default: ""},
			{Name: "snmp_v3_priv_passphrase", Type: Text{}, NotNull: true, Default: ""},
		},
		PrimaryKey: &PrimaryKey{Name: "snmp_monitors_pkey", Cols: []string{"id"}},
		Uniques: []Unique{
			{Name: "unique_snmp_monitor", Cols: []string{"name", "owner"}},
			{Name: "unique_snmp_address", Cols: []string{"udp_address", "owner"}},
		},
	}); err != nil {
		return fmt.Errorf(`failed to create the "snmp_monitors" table: %w`, err)
	}

	return nil
}

func ver0_10_0AddSNMPMonitorsDown(db Actions) error {
	if err := db.DropTable("snmp_monitors"); err != nil {
		return fmt.Errorf(`failed to drop the "snmp_monitors" table: %w`, err)
	}

	return nil
}

func ver0_10_0AddLocalAccountIPAddrUp(db Actions) error {
	if err := db.AlterTable("local_accounts",
		AddColumn{Name: "ip_addresses", Type: Text{}, NotNull: true, Default: ""},
	); err != nil {
		return fmt.Errorf(`failed to add the local account "ip_addresses" column: %w`, err)
	}

	return nil
}

func ver0_10_0AddLocalAccountIPAddrDown(db Actions) error {
	if err := db.AlterTable("local_accounts",
		DropColumn{Name: "ip_addresses"},
	); err != nil {
		return fmt.Errorf(`failed to drop the local account "ip_addresses" column: %w`, err)
	}

	return nil
}

func ver0_10_0AddTransferStartIndexUp(db Actions) error {
	if err := db.CreateIndex(&Index{
		Name:   "transfer_start_index",
		Unique: false,
		On:     "transfers",
		Cols:   []string{"start"},
	}); err != nil {
		return fmt.Errorf(`failed to create the "transfer_start_index" index: %w`, err)
	}

	if err := db.CreateIndex(&Index{
		Name:   "history_start_index",
		Unique: false,
		On:     "transfer_history",
		Cols:   []string{"start"},
	}); err != nil {
		return fmt.Errorf(`failed to create the "history_start_index" index: %w`, err)
	}

	return nil
}

func ver0_10_0AddTransferStartIndexDown(db Actions) error {
	if err := db.DropIndex("transfer_start_index", "transfers"); err != nil {
		return fmt.Errorf(`failed to drop the "transfer_start_index" index: %w`, err)
	}

	if err := db.DropIndex("history_start_index", "transfer_history"); err != nil {
		return fmt.Errorf(`failed to drop the "history_start_index" index: %w`, err)
	}

	return nil
}
