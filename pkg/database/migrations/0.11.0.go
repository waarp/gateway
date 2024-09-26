package migrations

import "fmt"

func ver0_11_0AddSNMPServerConfigUp(db Actions) error {
	if err := db.CreateTable("snmp_server_conf", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "owner", Type: Varchar(100), NotNull: true},
			{Name: "local_udp_address", Type: Varchar(255), NotNull: true},
			{Name: "community", Type: Varchar(100), NotNull: true, Default: "public"},
			{Name: "v3_only", Type: Boolean{}, NotNull: true, Default: false},
			{Name: "v3_username", Type: Varchar(100), NotNull: true},
			{Name: "v3_auth_protocol", Type: Varchar(50), NotNull: true},
			{Name: "v3_auth_passphrase", Type: Text{}, NotNull: true},
			{Name: "v3_priv_protocol", Type: Varchar(50), NotNull: true},
			{Name: "v3_priv_passphrase", Type: Text{}, NotNull: true},
		},
		PrimaryKey: &PrimaryKey{Name: "snmp_server_conf_pkey", Cols: []string{"id"}},
		Uniques: []Unique{
			{Name: "unique_snmp_server_conf", Cols: []string{"owner"}},
		},
	}); err != nil {
		return fmt.Errorf(`failed to create the "snmp_server_conf" table: %w`, err)
	}

	return nil
}

func ver0_11_0AddSNMPServerConfigDown(db Actions) error {
	if err := db.DropTable("snmp_server_conf"); err != nil {
		return fmt.Errorf(`failed to drop the "snmp_server_conf" table: %w`, err)
	}

	return nil
}
