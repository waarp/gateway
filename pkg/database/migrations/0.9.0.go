package migrations

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/compatibility"
)

func ver0_9_0AddCloudInstancesUp(db Actions) error {
	if err := db.CreateTable("cloud_instances", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "owner", Type: Varchar(100), NotNull: true},
			{Name: "name", Type: Varchar(100), NotNull: true},
			{Name: "type", Type: Varchar(50), NotNull: true},
			{Name: "api_key", Type: Text{}, NotNull: true, Default: ""},
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

func ver0_9_0AddCloudInstancesDown(db Actions) error {
	if err := db.DropTable("cloud_instances"); err != nil {
		return fmt.Errorf(`failed to drop the "cloud_instances" table: %w`, err)
	}

	return nil
}

func _ver0_9_0LocalPathToURLGetTransfers(db Actions, limit, offset int,
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

func _ver0_9_0LocalPathToURLToURL(path string) string {
	path = filepath.ToSlash(path)
	path = strings.TrimLeft(path, "/")
	path = "file:/" + path

	return path
}

func _ver0_9_0LocalPathToURLFromURL(path string) string {
	path = strings.TrimPrefix(path, "file:/")
	path = filepath.FromSlash(path)

	if !filepath.IsAbs(path) {
		path = "/" + path
	}

	return path
}

func ver0_9_0LocalPathToURLUp(db Actions) error {
	const limit = 20

	for offset := 0; ; offset += limit {
		transfers, getErr := _ver0_9_0LocalPathToURLGetTransfers(db, limit, offset)
		if getErr != nil {
			return getErr
		} else if len(transfers) == 0 {
			break
		}

		for _, trans := range transfers {
			newPath := _ver0_9_0LocalPathToURLToURL(trans.path)
			if err := db.Exec(`UPDATE transfers SET local_path=? WHERE id=?`,
				newPath, trans.id); err != nil {
				return fmt.Errorf(`failed to update the "transfers" table: %w`, err)
			}
		}
	}

	return nil
}

func ver0_9_0LocalPathToURLDown(db Actions) error {
	const limit = 20

	for offset := 0; ; offset += limit {
		transfers, getErr := _ver0_9_0LocalPathToURLGetTransfers(db, limit, offset)
		if getErr != nil {
			return getErr
		} else if len(transfers) == 0 {
			break
		}

		for _, trans := range transfers {
			oldPath := _ver0_9_0LocalPathToURLFromURL(trans.path)
			if err := db.Exec(`UPDATE transfers SET local_path=? WHERE id=?`,
				oldPath, trans.id); err != nil {
				return fmt.Errorf(`failed to update the "transfers" table: %w`, err)
			}
		}
	}

	return nil
}

func ver0_9_0FixLocalServerEnabledUp(db Actions) error {
	if db.GetDialect() == SQLite {
		// Due to how SQLite handles ALTER TABLE, we must drop the "normalized_transfers"
		// view before altering the table, otherwise the operation fails.
		if err := ver0_7_0AddNormalizedTransfersViewDown(db); err != nil {
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
		if err := ver0_7_0AddNormalizedTransfersViewUp(db); err != nil {
			return err
		}
	}

	return nil
}

func ver0_9_0FixLocalServerEnabledDown(db Actions) error {
	if db.GetDialect() == SQLite {
		// Due to how SQLite handles ALTER TABLE, we must drop the "normalized_transfers"
		// view before altering the table, otherwise the operation fails.
		if err := ver0_7_0AddNormalizedTransfersViewDown(db); err != nil {
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
		if err := ver0_7_0AddNormalizedTransfersViewUp(db); err != nil {
			return err
		}
	}

	return nil
}

func ver0_9_0AddClientsTableUp(db Actions) error {
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
		SELECT DISTINCT owner,protocol AS new_name,protocol FROM remote_agents
		INNER JOIN users ON true ORDER BY protocol, owner`); err != nil {
		return fmt.Errorf("failed to insert the default client: %w", err)
	}

	return nil
}

func ver0_9_0AddClientsTableDown(db Actions) error {
	if err := db.DropTable("clients"); err != nil {
		return fmt.Errorf("failed to drop the 'clients' table: %w", err)
	}

	return nil
}

func ver0_9_0AddRemoteAgentOwnerUp(db Actions) error {
	// We drop the "normalized_transfers" view, otherwise the ALTER TABLE fails
	// on SQLite. It is restored in a later script.
	if err := ver0_7_0AddNormalizedTransfersViewDown(db); err != nil {
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

func ver0_9_0AddRemoteAgentOwnerDown(db Actions) error {
	if err := db.AlterTable("remote_agents",
		DropColumn{Name: "owner"},
		AddUnique{Name: "unique_remote_agent", Cols: []string{"name"}},
	); err != nil {
		return fmt.Errorf("failed to alter the 'remote_agents' table: %w", err)
	}

	if err := ver0_7_0AddNormalizedTransfersViewUp(db); err != nil {
		return err
	}

	return nil
}

func _ver0_9_0DuplicateRemoteAgentsDuplicateCrypto(db Actions,
	idCol string, oldOwnerID int64, newIDCte string,
) error {
	type crypto struct{ name, pk, cert, sshKey string }

	var cryptos []*crypto

	if err := func() error {
		rows, queryErr := db.Query(fmt.Sprintf(
			`SELECT name,private_key,certificate,ssh_public_key 
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

func _ver0_9_0DuplicateRemoteAgentsDuplicateRuleAccess(db Actions,
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

func _ver0_9_0DuplicateRemoteAgentsDuplicateRemoteAccounts(db Actions,
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

		if err := _ver0_9_0DuplicateRemoteAgentsDuplicateCrypto(db,
			"remote_account_id", remAcc.id, newIDCte); err != nil {
			return err
		}

		if err := _ver0_9_0DuplicateRemoteAgentsDuplicateRuleAccess(db,
			"remote_account_id", remAcc.id, newIDCte); err != nil {
			return err
		}
	}

	return nil
}

func _ver0_9_0DuplicateRemoteAgentsDuplicateRemoteAgents(db Actions,
	sourceOwner, newOwner string,
) error {
	type remoteAgent struct {
		id                           int64
		name, proto, addr, protoConf string
	}

	var remoteAgents []*remoteAgent

	if err := func() error {
		rows, queryErr := db.Query(`SELECT id,name,protocol,address,proto_config
		    FROM remote_agents WHERE owner=? ORDER BY id`, sourceOwner)
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

		if err := _ver0_9_0DuplicateRemoteAgentsDuplicateCrypto(db,
			"remote_agent_id", remAg.id, newIDCte); err != nil {
			return err
		}

		if err := _ver0_9_0DuplicateRemoteAgentsDuplicateRuleAccess(db,
			"remote_agent_id", remAg.id, newIDCte); err != nil {
			return err
		}

		if err := _ver0_9_0DuplicateRemoteAgentsDuplicateRemoteAccounts(db,
			remAg.id, newIDCte); err != nil {
			return err
		}
	}

	return nil
}

func _ver0_9_0DuplicateRemoteAgentsGetOtherOwners(db Actions, primaryOwner string,
) ([]string, error) {
	var owners []string

	rows, queryErr := db.Query(`SELECT DISTINCT owner FROM users WHERE owner<>?
		ORDER BY owner`, primaryOwner)
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

func _ver0_9_0DuplicateRemoteAgentsGetFirstOwner(db Actions) (string, error) {
	var owner string

	row := db.QueryRow(`SELECT owner FROM users ORDER BY username LIMIT 1`)
	if err := row.Scan(&owner); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}

		return "", fmt.Errorf("failed to retrieve the primary owner: %w", err)
	}

	return owner, nil
}

func ver0_9_0DuplicateRemoteAgentsUp(db Actions) error {
	firstOwner, firstErr := _ver0_9_0DuplicateRemoteAgentsGetFirstOwner(db)
	if firstErr != nil {
		return firstErr
	} else if firstOwner == "" {
		return nil // nothing to do
	}

	if err := db.Exec(`UPDATE remote_agents SET owner=?`, firstOwner); err != nil {
		return fmt.Errorf("failed to update remote agents: %w", err)
	}

	owners, ownErr := _ver0_9_0DuplicateRemoteAgentsGetOtherOwners(db, firstOwner)
	if ownErr != nil {
		return ownErr
	}

	for _, owner := range owners {
		if err := _ver0_9_0DuplicateRemoteAgentsDuplicateRemoteAgents(db, firstOwner, owner); err != nil {
			return err
		}
	}

	return nil
}

func ver0_9_0DuplicateRemoteAgentsDown(db Actions) error {
	firstOwner, firstErr := _ver0_9_0DuplicateRemoteAgentsGetFirstOwner(db)
	if firstErr != nil {
		return firstErr
	} else if firstOwner == "" {
		return nil // nothing to do
	}

	if err := db.Exec(`DELETE FROM remote_agents WHERE owner<>?`, firstOwner); err != nil {
		return fmt.Errorf("failed to delete the duplicated remote agents: %w", err)
	}

	if err := db.Exec(`UPDATE remote_agents SET owner=''`); err != nil {
		return fmt.Errorf("failed to update remote agents: %w", err)
	}

	return nil
}

func _ver0_9_0RelinkTransfersGetPartners(db Actions) ([][4]any, error,
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

func ver0_9_0RelinkTransfersUp(db Actions) error {
	partners, cliErr := _ver0_9_0RelinkTransfersGetPartners(db)
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

func ver0_9_0RelinkTransfersDown(db Actions) error {
	primaryOwner, primaryErr := _ver0_9_0DuplicateRemoteAgentsGetFirstOwner(db)
	if primaryErr != nil {
		return primaryErr
	}

	partners, cliErr := _ver0_9_0RelinkTransfersGetPartners(db)
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
			WHERE remote_account_id=?`, login, name, primaryOwner, oldID); err != nil {
			return fmt.Errorf("failed to revert the transfer partners: %w", err)
		}
	}

	return nil
}

func ver0_9_0AddTransferClientIDUp(db Actions) error {
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

func ver0_9_0AddTransferClientIDDown(db Actions) error {
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

func ver0_9_0AddHistoryClientUp(db Actions) error {
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

func ver0_9_0AddHistoryClientDown(db Actions) error {
	if err := db.AlterTable("transfer_history",
		DropColumn{Name: "client"}); err != nil {
		return fmt.Errorf("failed to alter the 'transfer_history' table: %w", err)
	}

	return nil
}

func ver0_9_0RestoreNormalizedTransfersViewUp(db Actions) error {
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

func ver0_9_0RestoreNormalizedTransfersViewDown(db Actions) error {
	if err := db.DropView("normalized_transfers"); err != nil {
		return fmt.Errorf("failed to drop the normalized transfer view: %w", err)
	}

	return nil
}

func ver0_9_0AddCredTableUp(db Actions) error {
	//nolint:gomnd //magic numbers are required here for variable length types
	if err := db.CreateTable("credentials", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "local_agent_id", Type: BigInt{}},
			{Name: "remote_agent_id", Type: BigInt{}},
			{Name: "local_account_id", Type: BigInt{}},
			{Name: "remote_account_id", Type: BigInt{}},
			{Name: "name", Type: Varchar(100), NotNull: true},
			{Name: "type", Type: Varchar(50), NotNull: true},
			{Name: "value", Type: Text{}, NotNull: true},
			{Name: "value2", Type: Text{}, NotNull: true, Default: ""},
		},
		PrimaryKey: &PrimaryKey{Name: "credentials_pkey", Cols: []string{"id"}},
		ForeignKeys: []ForeignKey{{
			Name: "local_agent_id_fkey", Cols: []string{"local_agent_id"},
			RefTbl: "local_agents", RefCols: []string{"id"},
			OnUpdate: Restrict, OnDelete: Cascade,
		}, {
			Name: "remote_agent_id_fkey", Cols: []string{"remote_agent_id"},
			RefTbl: "remote_agents", RefCols: []string{"id"},
			OnUpdate: Restrict, OnDelete: Cascade,
		}, {
			Name: "local_account_id_fkey", Cols: []string{"local_account_id"},
			RefTbl: "local_accounts", RefCols: []string{"id"},
			OnUpdate: Restrict, OnDelete: Cascade,
		}, {
			Name: "remote_account_id_fkey", Cols: []string{"remote_account_id"},
			RefTbl: "remote_accounts", RefCols: []string{"id"},
			OnUpdate: Restrict, OnDelete: Cascade,
		}},
		Uniques: []Unique{
			{Name: "auth_local_agent_unique", Cols: []string{"name", "local_agent_id"}},
			{Name: "auth_remote_agent_unique", Cols: []string{"name", "remote_agent_id"}},
			{Name: "auth_local_account_unique", Cols: []string{"name", "local_account_id"}},
			{Name: "auth_remote_account_unique", Cols: []string{"name", "remote_account_id"}},
		},
		Checks: []Check{{
			Name: "auth_owner_check",
			Expr: checkOnlyOneNotNull("local_agent_id", "remote_agent_id",
				"local_account_id", "remote_account_id"),
		}},
	}); err != nil {
		return fmt.Errorf("failed to create the 'credentials' table: %w", err)
	}

	return nil
}

func ver0_9_0AddCredTableDown(db Actions) error {
	if err := db.DropTable("credentials"); err != nil {
		return fmt.Errorf("failed to drop the 'credentials' table: %w", err)
	}

	return nil
}

//nolint:gochecknoglobals //global var is needed here for tests
var _ver0_9_0FillAuthTableIgnoreCertParseError bool

func _ver0_9_0FillAuthTableGetCerts(db Actions) ([]struct {
	id    int64
	value string
}, error,
) {
	rows, queryErr := db.Query(`SELECT id,value FROM credentials 
		WHERE type='tls_certificate' OR type='trusted_tls_certificate'`)
	if queryErr != nil {
		return nil, fmt.Errorf("failed to query the 'credentials' table: %w", queryErr)
	}

	defer rows.Close()

	var certs []struct {
		id    int64
		value string
	}

	for rows.Next() {
		cert := struct {
			id    int64
			value string
		}{}

		if scanErr := rows.Scan(&cert.id, &cert.value); scanErr != nil {
			return nil, fmt.Errorf("failed to scan the 'credentials' table rows: %w", scanErr)
		}

		certs = append(certs, cert)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate over the 'credentials' table rows: %w", err)
	}

	return certs, nil
}

func ver0_9_0FillCredTableUp(db Actions) error {
	if err := db.Exec(`INSERT INTO 
		credentials (local_account_id, name, type, value)
		SELECT id, 'password', 'password_hash', password_hash FROM local_accounts
		WHERE password_hash IS NOT NULL AND LENGTH(password_hash) > 0`); err != nil {
		return fmt.Errorf("failed to add the local account passwords to the 'credentials' table: %w", err)
	}

	trimPassword := ltrim(db, "'$AES$'", "password")
	if err := db.Exec(`INSERT INTO 
		credentials (remote_account_id, name, type, value)
		SELECT id, 'password', 'password', ` + trimPassword + ` FROM remote_accounts
		WHERE password IS NOT NULL AND LENGTH(password) > 0`); err != nil {
		return fmt.Errorf("failed to add the remote account passwords to the 'credentials' table: %w", err)
	}

	if err := db.Exec(`INSERT INTO
		credentials(remote_agent_id, local_account_id, name, type, value)
		SELECT remote_agent_id, local_account_id, name, 'trusted_tls_certificate', certificate
		FROM crypto_credentials
		WHERE (remote_agent_id IS NOT NULL OR local_account_id IS NOT NULL)
			AND (certificate IS NOT NULL AND LENGTH(certificate) > 0)`); err != nil {
		return fmt.Errorf("failed to add the remote TLS certificates to the 'credentials' table: %w", err)
	}

	trimPK := ltrim(db, "'$AES$'", "private_key")
	if err := db.Exec(`INSERT INTO
		credentials(local_agent_id, remote_account_id, name, type, value, value2)
		SELECT local_agent_id, remote_account_id, name, 'tls_certificate',
			certificate, ` + trimPK + ` FROM crypto_credentials
		WHERE (local_agent_id IS NOT NULL OR remote_account_id IS NOT NULL)
			AND (certificate IS NOT NULL AND LENGTH(certificate) > 0)`); err != nil {
		return fmt.Errorf("failed to add the local TLS certificates to the 'credentials' table: %w", err)
	}

	if err := db.Exec(`INSERT INTO
		credentials(remote_agent_id, local_account_id, name, type, value)
		SELECT remote_agent_id, local_account_id, name, 'ssh_public_key', ssh_public_key
		FROM crypto_credentials
		WHERE (remote_agent_id IS NOT NULL OR local_account_id IS NOT NULL)
			AND (ssh_public_key IS NOT NULL AND LENGTH(ssh_public_key) > 0)`); err != nil {
		return fmt.Errorf("failed to add the SSH public keys to the 'credentials' table: %w", err)
	}

	if err := db.Exec(`INSERT INTO
		credentials(local_agent_id, remote_account_id, name, type, value)
		SELECT local_agent_id, remote_account_id, name, 'ssh_private_key', ` + trimPK + `
		FROM crypto_credentials
		WHERE (local_agent_id IS NOT NULL OR remote_account_id IS NOT NULL)
			AND (private_key IS NOT NULL AND LENGTH(private_key) > 0) 
			AND (certificate IS NULL OR LENGTH(certificate) = 0)`); err != nil {
		return fmt.Errorf("failed to add the SSH private keys to the credentials table: %w", err)
	}

	certs, certsErr := _ver0_9_0FillAuthTableGetCerts(db)
	if certsErr != nil {
		return certsErr
	}

	for _, cert := range certs {
		if certs, err := utils.ParsePEMCertChain(cert.value); err != nil {
			if !_ver0_9_0FillAuthTableIgnoreCertParseError {
				return fmt.Errorf("failed to parse the TLS certificate credentials: %w", err)
			}
		} else if compatibility.IsLegacyR66Cert(certs[0]) {
			if err2 := db.Exec(`UPDATE credentials SET type='r66_legacy_certificate',
				value='', value2='' WHERE id=?`, cert.id); err2 != nil {
				return fmt.Errorf("failed to update the legacy R66 certificate credentials: %w", err2)
			}
		}
	}

	return nil
}

func ver0_9_0FillCredTableDown(db Actions) error {
	if err := db.Exec(`DELETE FROM credentials`); err != nil {
		return fmt.Errorf("failed to delete the data from table 'crypto_credentials': %w", err)
	}

	return nil
}

func ver0_9_0RemoveOldAuthsUp(db Actions) error {
	if err := db.AlterTable("local_accounts", DropColumn{Name: "password_hash"}); err != nil {
		return fmt.Errorf("failed to drop the local account 'password_hash' column: %w", err)
	}

	if err := db.AlterTable("remote_accounts", DropColumn{Name: "password"}); err != nil {
		return fmt.Errorf("failed to drop the remote account 'password' column: %w", err)
	}

	if err := db.DropTable("crypto_credentials"); err != nil {
		return fmt.Errorf("failed to drop the 'crypto_credentials' table: %w", err)
	}

	return nil
}

func _ver0_9_0RemoveOldAuthsRecreateCryptoCredentialsTable(db Actions) error {
	//nolint:gomnd,dupl //magic numbers are required here for variable length types
	if err := db.CreateTable("crypto_credentials", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "name", Type: Varchar(100), NotNull: true},
			{Name: "local_agent_id", Type: BigInt{}},
			{Name: "remote_agent_id", Type: BigInt{}},
			{Name: "local_account_id", Type: BigInt{}},
			{Name: "remote_account_id", Type: BigInt{}},
			{Name: "private_key", Type: Text{}, NotNull: true, Default: ""},
			{Name: "certificate", Type: Text{}, NotNull: true, Default: ""},
			{Name: "ssh_public_key", Type: Text{}, NotNull: true, Default: ""},
		},
		PrimaryKey: &PrimaryKey{Name: "crypto_credentials_pkey", Cols: []string{"id"}},
		ForeignKeys: []ForeignKey{{
			Name: "local_agent_fkey", Cols: []string{"local_agent_id"},
			RefTbl: "local_agents", RefCols: []string{"id"},
			OnUpdate: Restrict, OnDelete: Cascade,
		}, {
			Name: "remote_agent_fkey", Cols: []string{"remote_agent_id"},
			RefTbl: "remote_agents", RefCols: []string{"id"},
			OnUpdate: Restrict, OnDelete: Cascade,
		}, {
			Name: "local_account_fkey", Cols: []string{"local_account_id"},
			RefTbl: "local_accounts", RefCols: []string{"id"},
			OnUpdate: Restrict, OnDelete: Cascade,
		}, {
			Name: "remote_account_fkey", Cols: []string{"remote_account_id"},
			RefTbl: "remote_accounts", RefCols: []string{"id"},
			OnUpdate: Restrict, OnDelete: Cascade,
		}},
		Uniques: []Unique{
			{Name: "unique_crypto_loc_agent", Cols: []string{"name", "local_agent_id"}},
			{Name: "unique_crypto_rem_agent", Cols: []string{"name", "remote_agent_id"}},
			{Name: "unique_crypto_loc_account", Cols: []string{"name", "local_account_id"}},
			{Name: "unique_crypto_rem_account", Cols: []string{"name", "remote_account_id"}},
		},
		Checks: []Check{{
			Name: "crypto_check_owner",
			Expr: checkOnlyOneNotNull("local_agent_id", "remote_agent_id",
				"local_account_id", "remote_account_id"),
		}},
	}); err != nil {
		return fmt.Errorf("failed to recreate the dropped crypto_credentials table: %w", err)
	}

	return nil
}

func ver0_9_0RemoveOldAuthsDown(db Actions) error {
	if err := _ver0_9_0RemoveOldAuthsRecreateCryptoCredentialsTable(db); err != nil {
		return err
	}

	if err := db.Exec(`INSERT INTO crypto_credentials (local_agent_id, remote_agent_id,
		local_account_id, remote_account_id, name, certificate, private_key) 
		SELECT local_agent_id, remote_agent_id, local_account_id, remote_account_id, name, 
		value, value2 FROM credentials WHERE type = 'tls_certificate'
		OR type = 'trusted_tls_certificate'`); err != nil {
		return fmt.Errorf("failed to restore the TLS certificates: %w", err)
	}

	if err := db.Exec(`INSERT INTO crypto_credentials (local_agent_id, remote_account_id, 
		name, private_key) SELECT local_agent_id, remote_account_id, name, value
		FROM credentials WHERE type = 'ssh_private_key'`); err != nil {
		return fmt.Errorf("failed to restore the SSH private keys: %w", err)
	}

	if err := db.Exec(`INSERT INTO crypto_credentials (remote_agent_id, local_account_id, 
		name, ssh_public_key) SELECT remote_agent_id, local_account_id, name, value
		FROM credentials WHERE type = 'ssh_public_key'`); err != nil {
		return fmt.Errorf("failed to restore the SSH public keys: %w", err)
	}

	if err := db.Exec(`UPDATE crypto_credentials SET private_key=` +
		concat(db, "'$AES$'", "private_key") + ` WHERE private_key<>''`); err != nil {
		return fmt.Errorf("failed to prefix the private keys: %w", err)
	}

	if err := db.Exec(`INSERT INTO crypto_credentials 
		(local_agent_id, remote_account_id, name, certificate, private_key) 
		SELECT local_agent_id, remote_account_id, name,	?, ? FROM credentials
		WHERE type = 'r66_legacy_certificate' AND 
			(local_agent_id IS NOT NULL OR remote_account_id IS NOT NULL)`,
		compatibility.LegacyR66CertPEM, "$AES$"+compatibility.LegacyR66KeyPEM); err != nil {
		return fmt.Errorf("failed to restore the local legacy R66 certificates: %w", err)
	}

	if err := db.Exec(`INSERT INTO crypto_credentials 
		(remote_agent_id, local_account_id,	name, certificate) 
		SELECT remote_agent_id, local_account_id, name, ? FROM credentials 
		WHERE type = 'r66_legacy_certificate' AND
			(remote_agent_id IS NOT NULL OR local_account_id IS NOT NULL)`,
		compatibility.LegacyR66CertPEM); err != nil {
		return fmt.Errorf("failed to restore the remote legacy R66 certificates: %w", err)
	}

	if err := db.AlterTable("local_accounts",
		AddColumn{Name: "password_hash", Type: Text{}, NotNull: true, Default: ""},
	); err != nil {
		return fmt.Errorf("failed to restore the local account 'password_hash' column: %w", err)
	}

	if err := db.Exec(`UPDATE local_accounts SET password_hash = ` + ifNull(db,
		`SELECT credentials.value FROM local_accounts LEFT JOIN credentials
		ON credentials.local_account_id = local_accounts.id 
		WHERE credentials.type = 'password'`, "''")); err != nil {
		return fmt.Errorf("failed to restore the local account 'password_hash' values: %w", err)
	}

	if err := db.AlterTable("remote_accounts",
		AddColumn{Name: "password", Type: Text{}, NotNull: true, Default: ""},
	); err != nil {
		return fmt.Errorf("failed to restore the remote account 'password' column: %w", err)
	}

	if err := db.Exec(`UPDATE remote_accounts SET password = ` + ifNull(db,
		concat(db, `'$AES$'`, `(SELECT credentials.value FROM remote_accounts
		LEFT JOIN credentials ON credentials.remote_account_id = remote_accounts.id
		WHERE credentials.type = 'password')`), "''")); err != nil {
		return fmt.Errorf("failed to restore the remote account 'password' values: %w", err)
	}

	return nil
}

func _ver0_9_0MoveR66ServerAuthDo(db Actions, tbl, col, authType string) error {
	r66Ags, getErr := ver0_7_5SplitR66TLSgetR66AgentsList(db, tbl)
	if getErr != nil {
		return getErr
	}

	for _, ag := range r66Ags {
		if pwdInter, ok1 := ag.conf["serverPassword"]; ok1 {
			delete(ag.conf, "serverPassword")

			if pwd, ok2 := pwdInter.(string); ok2 && pwd != "" {
				if err := db.Exec(`INSERT INTO credentials (`+col+`, name,
					type, value) VALUES (?, 'password', ?, ?)`,
					ag.id, authType, strings.TrimPrefix(pwd, "$AES$")); err != nil {
					return fmt.Errorf("failed to insert the %s password: %w", tbl, err)
				}
			}

			rawConf, convErr := json.Marshal(ag.conf)
			if convErr != nil {
				return fmt.Errorf("failed to serialize the %s proto config: %w", tbl, convErr)
			}

			if err := db.Exec(fmt.Sprintf(`UPDATE %s SET proto_config=? WHERE id=?`, tbl),
				rawConf, ag.id); err != nil {
				return fmt.Errorf("failed to update the %s proto config: %w", tbl, err)
			}
		}
	}

	return nil
}

func _ver0_9_0MoveR66ServerAuthUndo(db Actions, tbl, col, authType string, chpwd func(string) string) error {
	r66Ags, err := ver0_7_5SplitR66TLSgetR66AgentsList(db, tbl)
	if err != nil {
		return err
	}

	for _, ag := range r66Ags {
		row := db.QueryRow(`SELECT value FROM credentials WHERE `+col+`=? AND type=?`,
			ag.id, authType)

		var pswd string
		if err := row.Scan(&pswd); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}

			return fmt.Errorf("failed to retrieve the %s's password: %w", tbl, err)
		}

		ag.conf["serverPassword"] = chpwd(pswd)

		rawConf, convErr := json.Marshal(ag.conf)
		if convErr != nil {
			return fmt.Errorf("failed to serialize the %s's proto config: %w", tbl, convErr)
		}

		if err := db.Exec(fmt.Sprintf(`UPDATE %s SET proto_config=? WHERE id=?`, tbl),
			rawConf, ag.id); err != nil {
			return fmt.Errorf("failed to update the %s password: %w", tbl, err)
		}
	}

	if err := db.Exec(`DELETE FROM credentials WHERE (type='password' OR type='password_hash')
	    AND ` + col + ` IN (SELECT id FROM ` + tbl + ` WHERE protocol='r66' OR protocol='r66-tls')`); err != nil {
		return fmt.Errorf("failed to delete the R66 credentials: %w", err)
	}

	return nil
}

func ver0_9_0MoveR66ServerPswdUp(db Actions) error {
	if err := _ver0_9_0MoveR66ServerAuthDo(db, "local_agents", "local_agent_id", "password"); err != nil {
		return err
	}

	return _ver0_9_0MoveR66ServerAuthDo(db, "remote_agents", "remote_agent_id", "password_hash")
}

func ver0_9_0MoveR66ServerPswdDown(db Actions) error {
	if err := _ver0_9_0MoveR66ServerAuthUndo(db, "local_agents", "local_agent_id", "password",
		func(s string) string { return "$AES$" + s },
	); err != nil {
		return err
	}

	return _ver0_9_0MoveR66ServerAuthUndo(db, "remote_agents", "remote_agent_id", "password_hash",
		func(s string) string { return s })
}

func ver0_9_0AddAuthoritiesTableUp(db Actions) error {
	if err := db.CreateTable("auth_authorities", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "name", Type: Varchar(100), NotNull: true},
			{Name: "type", Type: Varchar(50), NotNull: true},
			{Name: "public_identity", Type: Text{}, NotNull: true},
		},
		PrimaryKey: &PrimaryKey{Name: "auth_authorities_pkey", Cols: []string{"id"}},
		Uniques:    []Unique{{Name: "unique_authority_name", Cols: []string{"name"}}},
	}); err != nil {
		return fmt.Errorf("failed to create the authorities table: %w", err)
	}

	if err := db.CreateTable("authority_hosts", &Table{
		Columns: []Column{
			{Name: "authority_id", Type: BigInt{}, NotNull: true},
			{Name: "host", Type: Varchar(255), NotNull: true},
		},
		Uniques: []Unique{{Name: "unique_authority_host", Cols: []string{"authority_id", "host"}}},
		ForeignKeys: []ForeignKey{{
			Name: "authority_hosts_id_fkey", Cols: []string{"authority_id"},
			RefTbl: "auth_authorities", RefCols: []string{"id"},
			OnUpdate: Restrict, OnDelete: Cascade,
		}},
	}); err != nil {
		return fmt.Errorf("failed to create the authority hosts table: %w", err)
	}

	return nil
}

func ver0_9_0AddAuthoritiesTableDown(db Actions) error {
	if err := db.DropTable("authority_hosts"); err != nil {
		return fmt.Errorf("failed to drop the authority hosts table: %w", err)
	}

	if err := db.DropTable("auth_authorities"); err != nil {
		return fmt.Errorf("failed to drop the authorities table: %w", err)
	}

	return nil
}
