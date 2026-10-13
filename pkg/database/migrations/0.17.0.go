package migrations

import "fmt"

func ver0_17_0ChangeTransferInfoValueTypeUp(db Actions) error {
	if db.GetDialect() != MySQL {
		return nil
	}

	if err := db.ExecDDL("ALTER TABLE transfer_info MODIFY COLUMN `value` LONGTEXT " +
		"NOT NULL DEFAULT '{}'"); err != nil {
		return fmt.Errorf("failed to change transfer info value type: %w", err)
	}

	return nil
}

func ver0_17_0ChangeTransferInfoValueTypeDown(db Actions) error {
	if db.GetDialect() != MySQL {
		return nil
	}

	if err := db.ExecDDL("ALTER TABLE transfer_info MODIFY COLUMN `value` TEXT " +
		"NOT NULL DEFAULT '{}'"); err != nil {
		return fmt.Errorf("failed to change transfer info value type: %w", err)
	}

	return nil
}

func _ver0_17_0RemoveCloudCredsInOptsGetClouds(db Actions) ([]*struct {
	id   int64
	typ  string
	opts jsonObj
}, error,
) {
	type cloud = struct {
		id   int64
		typ  string
		opts jsonObj
	}

	rows, queryErr := db.Query("SELECT id,type,options FROM cloud_instances")
	if queryErr != nil {
		return nil, fmt.Errorf("failed to retrieve cloud instances: %w", queryErr)
	}

	defer rows.Close()

	var clouds []*cloud

	for rows.Next() {
		var c cloud

		if err := rows.Scan(&c.id, &c.typ, &c.opts); err != nil {
			return nil, fmt.Errorf("failed to scan cloud instance: %w", err)
		}

		clouds = append(clouds, &c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to retrieve cloud instances: %w", err)
	}

	return clouds, nil
}

func ver0_17_0RemoveCloudCredsInOptsUp(db Actions) error {
	clouds, dbErr := _ver0_17_0RemoveCloudCredsInOptsGetClouds(db)
	if dbErr != nil {
		return dbErr
	}

	for _, c := range clouds {
		switch c.typ {
		case "s3", "aws":
			delete(c.opts, "access_key_id")
			delete(c.opts, "secret_access_key")
		case "gcs", "google":
			delete(c.opts, "service_account_credentials")
			delete(c.opts, "service_account_file")
		case "onedrive", "sharepoint":
			delete(c.opts, "client_id")
			delete(c.opts, "client_secret")
		case "azblob", "azureblob", "azfiles", "azurefiles":
			delete(c.opts, "account")
			delete(c.opts, "key")
		default:
			//nolint:err113 //too specific
			return fmt.Errorf("unknown cloud instance type %q", c.typ)
		}

		if err := db.Exec("UPDATE cloud_instances SET options = ? WHERE id = ?",
			&c.opts, c.id); err != nil {
			return fmt.Errorf("failed to update cloud instance: %w", err)
		}
	}

	return nil
}
