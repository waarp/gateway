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
