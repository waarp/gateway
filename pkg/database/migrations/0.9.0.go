package migrations

import (
	"fmt"
	"path/filepath"
	"strings"
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
