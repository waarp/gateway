package migrations

import (
	"fmt"
	"path/filepath"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

type ver0_9_0AddCloudInstances struct{}

func (ver0_9_0AddCloudInstances) Up(db Actions) error {
	if err := db.CreateTable("cloud_instances", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "owner", Type: Varchar(100), NotNull: true},
			{Name: "name", Type: Varchar(100), NotNull: true},
			{Name: "type", Type: Varchar(50), NotNull: true},
			{Name: "key", Type: Text{}, NotNull: true, Default: ""},
			{Name: "secret", Type: Text{}, NotNull: true, Default: ""},
			{Name: "options", Type: Text{}, NotNull: true, Default: "{}"},
		},
		PrimaryKey: &PrimaryKey{Name: "cloud_instances_pkey", Cols: []string{"id"}},
		Uniques: []Unique{{
			Name: "unique_cloud_instance", Cols: []string{"owner", "name"},
		}},
	}); err != nil {
		return fmt.Errorf(`failed to create the "cloud_instances" table: %w`, err)
	}

	return nil
}

func (ver0_9_0AddCloudInstances) Down(db Actions) error {
	if err := db.DropTable("cloud_instances"); err != nil {
		return fmt.Errorf(`failed to drop the "cloud_instances" table: %w`, err)
	}

	return nil
}

type ver0_9_0LocalPathToURL struct{}

func (ver0_9_0LocalPathToURL) getTransfers(db Actions, limit, offset int,
) ([]*struct {
	id   int64
	path string
}, error,
) {
	type trans = struct {
		id   int64
		path string
	}

	rows, queryErr := db.Query(`SELECT id, local_path FROM transfers
		LIMIT ? OFFSET ?`, limit, offset)
	if queryErr != nil {
		return nil, fmt.Errorf(`failed to query the "transfers" table: %w`, queryErr)
	}

	defer rows.Close()

	transfers := make([]*trans, 0, limit)

	for rows.Next() {
		t := &trans{}
		if err := rows.Scan(&t.id, &t.path); err != nil {
			return nil, fmt.Errorf(`failed to scan the "transfers" table: %w`, err)
		}

		transfers = append(transfers, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(`failed to iterate over the "transfers" table: %w`, err)
	}

	return transfers, nil
}

func (ver0_9_0LocalPathToURL) toURL(path string) string {
	path = filepath.ToSlash(path)
	path = strings.TrimLeft(path, "/")
	path = "file:/" + path

	return path
}

func (ver0_9_0LocalPathToURL) fromURL(path string) string {
	path = strings.TrimPrefix(path, "file:/")
	path = filepath.FromSlash(path)

	if !filepath.IsAbs(path) {
		path = "/" + path
	}

	return path
}

func (v ver0_9_0LocalPathToURL) Up(db Actions) error {
	const limit = 20

	for offset := 0; ; offset += limit {
		transfers, getErr := v.getTransfers(db, limit, offset)
		if getErr != nil {
			return getErr
		} else if len(transfers) == 0 {
			break
		}

		for _, trans := range transfers {
			newPath := v.toURL(trans.path)
			if err := db.Exec(`UPDATE transfers SET local_path=? WHERE id=?`,
				newPath, trans.id); err != nil {
				return fmt.Errorf(`failed to update the "transfers" table: %w`, err)
			}
		}
	}

	return nil
}

func (v ver0_9_0LocalPathToURL) Down(db Actions) error {
	const limit = 20

	for offset := 0; ; offset += limit {
		transfers, getErr := v.getTransfers(db, limit, offset)
		if getErr != nil {
			return getErr
		} else if len(transfers) == 0 {
			break
		}

		for _, trans := range transfers {
			oldPath := v.fromURL(trans.path)
			if err := db.Exec(`UPDATE transfers SET local_path=? WHERE id=?`,
				oldPath, trans.id); err != nil {
				return fmt.Errorf(`failed to update the "transfers" table: %w`, err)
			}
		}
	}

	return nil
}

type ver0_9_0FixLocalServerEnabled struct{}

func (ver0_9_0FixLocalServerEnabled) Up(db Actions) error {
	if db.GetDialect() == SQLite {
		// Due to how SQLite handles ALTER TABLE, we must drop the "normalized_transfers"
		// view before altering the table, otherwise the operation fails.
		if err := (&ver0_7_0AddNormalizedTransfersView{}).Down(db); err != nil {
			return err
		}
	}

	if err := db.AlterTable("local_agents", AlterColumn{
		Name: "enabled", NewName: "disabled", Type: Boolean{},
		NotNull: true, Default: false,
	}); err != nil {
		return fmt.Errorf("failed to alter the 'local_agents' table: %w", err)
	}

	if err := db.Exec(`UPDATE local_agents SET disabled=(NOT disabled)`); err != nil {
		return fmt.Errorf("failed to update the server 'disabled' column: %w", err)
	}

	if db.GetDialect() == SQLite {
		// Now we restore the "normalized_transfers" view.
		if err := (&ver0_7_0AddNormalizedTransfersView{}).Up(db); err != nil {
			return err
		}
	}

	return nil
}

func (ver0_9_0FixLocalServerEnabled) Down(db Actions) error {
	if db.GetDialect() == SQLite {
		// Due to how SQLite handles ALTER TABLE, we must drop the "normalized_transfers"
		// view before altering the table, otherwise the operation fails.
		if err := (&ver0_7_0AddNormalizedTransfersView{}).Down(db); err != nil {
			return err
		}
	}

	if err := db.AlterTable("local_agents", AlterColumn{
		Name: "disabled", NewName: "enabled", Type: Boolean{},
		NotNull: true, Default: true,
	}); err != nil {
		return fmt.Errorf("failed to alter the 'local_agents' table: %w", err)
	}

	if err := db.Exec(`UPDATE local_agents SET enabled=(NOT enabled)`); err != nil {
		return fmt.Errorf("failed to update the server 'enabled' column: %w", err)
	}

	if db.GetDialect() == SQLite {
		// Now we restore the "normalized_transfers" view.
		if err := (&ver0_7_0AddNormalizedTransfersView{}).Up(db); err != nil {
			return err
		}
	}

	return nil
}

type ver0_9_0AddClientsTable struct{}

func (ver0_9_0AddClientsTable) Up(db Actions) error {
	if err := db.CreateTable("clients", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "owner", Type: Varchar(100), NotNull: true},
			{Name: "name", Type: Varchar(100), NotNull: true},
			{Name: "protocol", Type: Varchar(20), NotNull: true},
			{Name: "disabled", Type: Boolean{}, NotNull: true, Default: false},
			{Name: "local_address", Type: Varchar(255), NotNull: true, Default: ""},
			{Name: "proto_config", Type: Text{}, NotNull: true, Default: "{}"},
		},
		PrimaryKey: &PrimaryKey{Name: "clients_pkey", Cols: []string{"id"}},
		Uniques:    []Unique{{Name: "unique_client", Cols: []string{"name", "owner"}}},
	}); err != nil {
		return fmt.Errorf("failed to create the 'clients' table: %w", err)
	}

	if err := db.Exec(`INSERT INTO clients (owner,name,protocol) 
		SELECT DISTINCT owner,protocol,protocol FROM remote_agents
		INNER JOIN users ON true ORDER BY protocol, owner`); err != nil {
		return fmt.Errorf("failed to insert the default client: %w", err)
	}

	return nil
}

func (ver0_9_0AddClientsTable) Down(db Actions) error {
	if err := db.DropTable("clients"); err != nil {
		return fmt.Errorf("failed to drop the 'clients' table: %w", err)
	}

	return nil
}

type ver0_9_0AddRemoteAgentOwner struct{}

func (ver0_9_0AddRemoteAgentOwner) Up(db Actions) error {
	// We drop the "normalized_transfers" view, otherwise the ALTER TABLE fails
	// on SQLite. It is restored in a later script.
	if err := (&ver0_7_0AddNormalizedTransfersView{}).Down(db); err != nil {
		return err
	}

	if err := db.AlterTable("remote_agents",
		DropConstraint{Name: "unique_remote_agent"},
		AddColumn{Name: "owner", Type: Varchar(100) /*NotNull: true*/},
	); err != nil {
		return fmt.Errorf("failed to alter the 'remote_agents' table: %w", err)
	}

	return nil
}

func (ver0_9_0AddRemoteAgentOwner) Down(db Actions) error {
	if err := db.AlterTable("remote_agents",
		DropColumn{Name: "owner"},
		AddUnique{Name: "unique_remote_agent", Cols: []string{"name"}},
	); err != nil {
		return fmt.Errorf("failed to alter the 'remote_agents' table: %w", err)
	}

	if err := (&ver0_7_0AddNormalizedTransfersView{}).Up(db); err != nil {
		return err
	}

	return nil
}

type ver0_9_0DuplicateRemoteAgents struct{ currentOwner string }

func (v ver0_9_0DuplicateRemoteAgents) _duplicateCrypto(db Actions,
	idCol string, oldOwnerID int64, newIDCte string,
) error {
	type crypto struct{ name, pk, cert, sshKey string }

	var cryptos []*crypto

	if err := func() error {
		rows, queryErr := db.Query(fmt.Sprintf(`SELECT name,private_key,certificate,ssh_public_key 
			FROM crypto_credentials WHERE %s=? ORDER BY id`, idCol), oldOwnerID)
		if queryErr != nil {
			return fmt.Errorf("failed to retrieve the crypto credentials: %w", queryErr)
		}

		defer rows.Close()

		for rows.Next() {
			var crypt crypto
			if err := rows.Scan(&crypt.name, &crypt.pk, &crypt.cert, &crypt.sshKey); err != nil {
				return fmt.Errorf("failed to retrieve the crypto credential: %w", err)
			}

			cryptos = append(cryptos, &crypt)
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("failed to retrieve the crypto credentials: %w", err)
		}

		return nil
	}(); err != nil {
		return err
	}

	for _, crypt := range cryptos {
		if err := db.Exec(fmt.Sprintf(`INSERT INTO crypto_credentials
    			(%s,name,private_key,certificate,ssh_public_key)
				VALUES ((%s),?,?,?,?)`, idCol, newIDCte),
			crypt.name, crypt.pk, crypt.cert, crypt.sshKey); err != nil {
			return fmt.Errorf("failed to insert the new crypto credential: %w", err)
		}
	}

	return nil
}

func (v ver0_9_0DuplicateRemoteAgents) _duplicateRuleAccess(db Actions,
	idCol string, oldOwnerID int64, newIDCte string,
) error {
	var ruleIDs []int64

	if err := func() error {
		rows, queryErr := db.Query(fmt.Sprintf(`SELECT rule_id FROM rule_access 
               WHERE %s=? ORDER BY rule_id`, idCol), oldOwnerID)
		if queryErr != nil {
			return fmt.Errorf("failed to retrieve the rule accesses: %w", queryErr)
		}

		defer rows.Close()

		for rows.Next() {
			var ruleID int64
			if err := rows.Scan(&ruleID); err != nil {
				return fmt.Errorf("failed to retrieve the rule access: %w", err)
			}

			ruleIDs = append(ruleIDs, ruleID)
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("failed to retrieve the rule accesses: %w", err)
		}

		return nil
	}(); err != nil {
		return err
	}

	for _, ruleID := range ruleIDs {
		if err := db.Exec(fmt.Sprintf(`INSERT INTO rule_access(rule_id,%s)
				VALUES (?,(%s))`, idCol, newIDCte), ruleID); err != nil {
			return fmt.Errorf("failed to insert the new rule access: %w", err)
		}
	}

	return nil
}

func (v ver0_9_0DuplicateRemoteAgents) _duplicateRemoteAccounts(db Actions,
	oldAgentID int64, newAgentIDCte string,
) error {
	type remoteAccount struct {
		id              int64
		login, password string
	}

	var remoteAccounts []*remoteAccount

	if err := func() error {
		rows, queryErr := db.Query(`SELECT id,login,password FROM remote_accounts
			WHERE remote_agent_id=? ORDER BY id`, oldAgentID)
		if queryErr != nil {
			return fmt.Errorf("failed to retrieve the remote accounts: %w", queryErr)
		}

		defer rows.Close()

		for rows.Next() {
			var remAcc remoteAccount
			if err := rows.Scan(&remAcc.id, &remAcc.login, &remAcc.password); err != nil {
				return fmt.Errorf("failed to retrieve the remote account: %w", err)
			}

			remoteAccounts = append(remoteAccounts, &remAcc)
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("failed to retrieve the remote accounts: %w", err)
		}

		return nil
	}(); err != nil {
		return err
	}

	for _, remAcc := range remoteAccounts {
		if err := db.Exec(fmt.Sprintf(
			`INSERT INTO remote_accounts(remote_agent_id,login,password)
			VALUES ((%s),?,?)`, newAgentIDCte),
			remAcc.login, remAcc.password); err != nil {
			return fmt.Errorf("failed to insert the new remote account: %w", err)
		}

		newIDCte := fmt.Sprintf(`SELECT id FROM remote_accounts WHERE login='%s'
			AND remote_agent_id=(%s)`, remAcc.login, newAgentIDCte)

		if err := v._duplicateCrypto(db, "remote_account_id", remAcc.id, newIDCte); err != nil {
			return err
		}

		if err := v._duplicateRuleAccess(db, "remote_account_id", remAcc.id, newIDCte); err != nil {
			return err
		}
	}

	return nil
}

func (v ver0_9_0DuplicateRemoteAgents) _duplicateRemoteAgents(db Actions,
	newOwner string,
) error {
	type remoteAgent struct {
		id                           int64
		name, proto, addr, protoConf string
	}

	var remoteAgents []*remoteAgent

	if err := func() error {
		rows, queryErr := db.Query(`SELECT id,name,protocol,address,proto_config
		    FROM remote_agents WHERE owner=?`, v.currentOwner)
		if queryErr != nil {
			return fmt.Errorf("failed to retrieve the remote agents: %w", queryErr)
		}

		defer rows.Close()

		for rows.Next() {
			var remAg remoteAgent
			if err := rows.Scan(&remAg.id, &remAg.name, &remAg.proto, &remAg.addr,
				&remAg.protoConf); err != nil {
				return fmt.Errorf("failed to retrieve the remote agent: %w", err)
			}

			remoteAgents = append(remoteAgents, &remAg)
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("failed to retrieve the remote agents: %w", err)
		}

		return nil
	}(); err != nil {
		return err
	}

	for _, remAg := range remoteAgents {
		if err := db.Exec(
			`INSERT INTO remote_agents(name,owner,protocol,address,proto_config)
			VALUES (?,?,?,?,?)`,
			remAg.name, newOwner, remAg.proto, remAg.addr, remAg.protoConf); err != nil {
			return fmt.Errorf("failed to insert the new remote agent: %w", err)
		}

		newIDCte := fmt.Sprintf(`SELECT id FROM remote_agents WHERE name='%s' AND
			owner='%s'`, remAg.name, newOwner)

		if err := v._duplicateCrypto(db, "remote_agent_id", remAg.id, newIDCte); err != nil {
			return err
		}

		if err := v._duplicateRuleAccess(db, "remote_agent_id", remAg.id, newIDCte); err != nil {
			return err
		}

		if err := v._duplicateRemoteAccounts(db, remAg.id, newIDCte); err != nil {
			return err
		}
	}

	return nil
}

func (v ver0_9_0DuplicateRemoteAgents) _getOtherOwners(db Actions) ([]string, error) {
	var owners []string

	rows, queryErr := db.Query(`SELECT DISTINCT owner FROM users WHERE owner<>?
		ORDER BY owner`, v.currentOwner)
	if queryErr != nil {
		return nil, fmt.Errorf("failed to retrieve the owners: %w", queryErr)
	}

	defer rows.Close()

	for rows.Next() {
		var owner string
		if err := rows.Scan(&owner); err != nil {
			return nil, fmt.Errorf("failed to retrieve the owner: %w", err)
		}

		owners = append(owners, owner)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to retrieve the owners: %w", err)
	}

	return owners, nil
}

func (v ver0_9_0DuplicateRemoteAgents) Up(db Actions) error {
	if v.currentOwner == "" {
		v.currentOwner = conf.GlobalConfig.GatewayName
	}

	if err := db.Exec(`UPDATE remote_agents SET owner=?`, v.currentOwner); err != nil {
		return fmt.Errorf("failed to update remote agents: %w", err)
	}

	owners, ownErr := v._getOtherOwners(db)
	if ownErr != nil {
		return ownErr
	}

	for _, owner := range owners {
		if err := v._duplicateRemoteAgents(db, owner); err != nil {
			return err
		}
	}

	return nil
}

func (v ver0_9_0DuplicateRemoteAgents) Down(db Actions) error {
	if v.currentOwner == "" {
		v.currentOwner = conf.GlobalConfig.GatewayName
	}

	if err := db.Exec(`DELETE FROM remote_agents WHERE owner<>?`, v.currentOwner); err != nil {
		return fmt.Errorf("failed to delete the duplicated remote agents: %w", err)
	}

	if err := db.Exec(`UPDATE remote_agents SET owner=''`); err != nil {
		return fmt.Errorf("failed to update remote agents: %w", err)
	}

	return nil
}

type ver0_9_0RelinkTransfers struct{ currentOwner string }

func (ver0_9_0RelinkTransfers) getPartners(db Actions) ([][4]any, error,
) {
	rows, queryErr := db.Query(`SELECT DISTINCT a.id,a.login,p.name,t.owner FROM 
        transfers t INNER JOIN remote_accounts a ON t.remote_account_id=a.id 
        INNER JOIN remote_agents p ON a.remote_agent_id=p.id`)
	if queryErr != nil {
		return nil, fmt.Errorf("failed to retrieve the partners: %w", queryErr)
	}

	defer rows.Close()

	var partners [][4]any

	for rows.Next() {
		var (
			oldID              int64
			login, name, owner string
		)

		if err := rows.Scan(&oldID, &login, &name, &owner); err != nil {
			return nil, fmt.Errorf("failed to parse the partner values: %w", err)
		}

		partners = append(partners, [4]any{oldID, login, name, owner})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate the partner values: %w", err)
	}

	return partners, nil
}

func (v ver0_9_0RelinkTransfers) Up(db Actions) error {
	partners, cliErr := v.getPartners(db)
	if cliErr != nil {
		return cliErr
	}

	for _, partner := range partners {
		oldID := partner[0]
		login := partner[1]
		name := partner[2]
		owner := partner[3]

		if err := db.Exec(`UPDATE transfers SET
	               remote_account_id=(SELECT a.id FROM remote_accounts a
	                   INNER JOIN remote_agents p ON a.remote_agent_id=p.id
	                   WHERE a.login=? AND p.name=? AND p.owner=?)
	   			WHERE remote_account_id=? AND owner=?`,
			login, name, owner, oldID, owner); err != nil {
			return fmt.Errorf("failed to update the transfer partners: %w", err)
		}
	}

	return nil
}

func (v ver0_9_0RelinkTransfers) Down(db Actions) error {
	partners, cliErr := v.getPartners(db)
	if cliErr != nil {
		return cliErr
	}

	for _, partner := range partners {
		oldID := partner[0]
		login := partner[1]
		name := partner[2]

		if err := db.Exec(`UPDATE transfers SET
        	remote_account_id=(SELECT a.id FROM remote_accounts a
                INNER JOIN remote_agents p ON a.remote_agent_id=p.id
            	WHERE a.login=? AND p.name=? AND p.owner=?)
			WHERE remote_account_id=?`, login, name, v.currentOwner, oldID); err != nil {
			return fmt.Errorf("failed to revert the transfer partners: %w", err)
		}
	}

	return nil
}

type ver0_9_0AddTransferClientID struct{}

func (ver0_9_0AddTransferClientID) Up(db Actions) error {
	if err := db.AlterTable("transfers",
		AddColumn{Name: "client_id", Type: BigInt{}},
	); err != nil {
		return fmt.Errorf("failed to alter the 'transfers' table: %w", err)
	}

	checkExpr := "(remote_account_id IS NULL) = (client_id IS NULL)"
	query := `UPDATE transfers SET client_id=upd.cli_id FROM (
    		SELECT c.id AS cli_id, a.id AS acc_id FROM remote_accounts a 
    		    INNER JOIN remote_agents p ON a.remote_agent_id = p.id 
    	    	INNER JOIN clients c on c.owner = p.owner AND c.protocol = p.protocol
    		WHERE c.name = p.protocol
		) AS upd
		WHERE transfers.remote_account_id = upd.acc_id`

	if db.GetDialect() == MySQL {
		checkExpr = "IF(remote_account_id IS NULL, client_id IS NULL, client_id IS NOT NULL)"
		query = `UPDATE transfers, (
				SELECT c.id AS cli_id, a.id AS acc_id FROM remote_accounts a 
    			    INNER JOIN remote_agents p ON a.remote_agent_id = p.id 
    	    		INNER JOIN clients c on c.owner = p.owner AND c.protocol = p.protocol
				WHERE c.name = p.protocol) AS upd
			SET transfers.client_id = upd.cli_id
			WHERE transfers.remote_account_id = upd.acc_id`
	}

	if err := db.Exec(query); err != nil {
		return fmt.Errorf("failed to fill the transfer client_id column: %w", err)
	}

	if err := db.AlterTable("transfers",
		AddCheck{Name: "transfer_client_check", Expr: checkExpr},
	); err != nil {
		return fmt.Errorf("failed to alter the 'transfers' table: %w", err)
	}

	return nil
}

func (ver0_9_0AddTransferClientID) Down(db Actions) error {
	// The 2 modifications must be separated, otherwise it fails on SQLite.
	if err := db.AlterTable("transfers",
		DropConstraint{Name: "transfer_client_check"}); err != nil {
		return fmt.Errorf("failed to alter the 'transfers' table: %w", err)
	}

	if err := db.AlterTable("transfers",
		DropColumn{Name: "client_id"}); err != nil {
		return fmt.Errorf("failed to alter the 'transfers' table: %w", err)
	}

	return nil
}

type ver0_9_0AddHistoryClient struct{}

func (ver0_9_0AddHistoryClient) Up(db Actions) error {
	if err := db.AlterTable("transfer_history",
		AddColumn{Name: "client", Type: Varchar(100) /*NotNull: true*/}); err != nil {
		return fmt.Errorf("failed to alter the 'transfer_history' table: %w", err)
	}

	updQuery := `UPDATE transfer_history SET client=
    	(CASE WHEN is_server THEN '' ELSE protocol||'_client' END)`
	if db.GetDialect() == MySQL {
		updQuery = `UPDATE transfer_history SET client=
    		(CASE WHEN is_server THEN '' ELSE CONCAT(protocol,'_client') END)`
	}

	if err := db.Exec(updQuery); err != nil {
		return fmt.Errorf("failed to fill the history client column: %w", err)
	}

	if err := db.AlterTable("transfer_history",
		AlterColumn{Name: "client", Type: Varchar(100), NotNull: true}); err != nil {
		return fmt.Errorf("failed to alter the 'transfer_history' table: %w", err)
	}

	return nil
}

func (ver0_9_0AddHistoryClient) Down(db Actions) error {
	if err := db.AlterTable("transfer_history",
		DropColumn{Name: "client"}); err != nil {
		return fmt.Errorf("failed to alter the 'transfer_history' table: %w", err)
	}

	return nil
}

type ver0_9_0RestoreNormalizedTransfersView struct{}

func (ver0_9_0RestoreNormalizedTransfersView) Up(db Actions) error {
	transStop := utils.If(db.GetDialect() == PostgreSQL,
		"null::timestamp", "null")

	if err := db.CreateView(&View{
		Name: "normalized_transfers",
		As: `WITH transfers_as_history(id, owner, remote_transfer_id, is_server,
				is_send, rule, client, account, agent, protocol, src_filename, 
				dest_filename, local_path, remote_path, filesize, start, stop, 
				status, step, progress, task_number, error_code, error_details,
				is_transfer) AS (
					SELECT t.id, t.owner, t.remote_transfer_id, 
						t.local_account_id IS NOT NULL, r.is_send, r.name, 
						(CASE WHEN t.client_id IS NULL THEN '' ELSE c.name END),
						(CASE WHEN t.local_account_id IS NULL THEN ra.login ELSE la.login END),
						(CASE WHEN t.local_account_id IS NULL THEN p.name ELSE s.name END),
						(CASE WHEN t.local_account_id IS NULL THEN p.protocol ELSE s.protocol END),
						t.src_filename, t.dest_filename, t.local_path, t.remote_path, t.filesize,
						t.start, ` + transStop + `, t.status, t.step, t.progress, t.task_number,
						t.error_code, t.error_details, true
					FROM transfers AS t
					LEFT JOIN rules AS r ON t.rule_id = r.id
					LEFT JOIN clients AS c ON t.client_id = c.id
					LEFT JOIN local_accounts  AS la ON  t.local_account_id = la.id
					LEFT JOIN remote_accounts AS ra ON t.remote_account_id = ra.id
					LEFT JOIN local_agents    AS s ON la.local_agent_id = s.id 
					LEFT JOIN remote_agents   AS p ON ra.remote_agent_id = p.id
				)
			SELECT id, owner, remote_transfer_id, is_server, is_send, rule, client,
		        account, agent, protocol, src_filename, dest_filename, local_path,
				remote_path, filesize, start, stop, status, step, progress, 
				task_number, error_code, error_details, false AS is_transfer
			FROM transfer_history UNION
			SELECT * FROM transfers_as_history`,
	}); err != nil {
		return fmt.Errorf("failed to create the normalized transfer view: %w", err)
	}

	return nil
}

func (ver0_9_0RestoreNormalizedTransfersView) Down(db Actions) error {
	if err := db.DropView("normalized_transfers"); err != nil {
		return fmt.Errorf("failed to drop the normalized transfer view: %w", err)
	}

	return nil
}
