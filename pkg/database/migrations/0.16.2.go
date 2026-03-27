package migrations

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/ebicsbtfseed"
)

type ver0_16_2StandardBTFCatalogSeed struct {
	Name           string                          `json:"name"`
	Scope          string                          `json:"scope"`
	CatalogVersion string                          `json:"catalogVersion"`
	SourceType     string                          `json:"sourceType"`
	SourceRef      string                          `json:"sourceRef"`
	Status         string                          `json:"status"`
	Entries        []ver0_16_2StandardBTFEntrySeed `json:"entries"`
}

type ver0_16_2StandardBTFEntrySeed struct {
	EntryKey          string         `json:"entryKey"`
	OrderType         string         `json:"orderType"`
	Direction         string         `json:"direction"`
	ServiceName       string         `json:"serviceName"`
	ServiceOption     string         `json:"serviceOption"`
	Scope             string         `json:"scope"`
	MsgName           string         `json:"msgName"`
	ContainerType     string         `json:"containerType"`
	CountryGroup      string         `json:"countryGroup"`
	IsDefaultTemplate bool           `json:"isDefaultTemplate"`
	Metadata          map[string]any `json:"metadata"`
}

func ver0_16_2SeedEbicsStandardBTFCatalogsUp(db Actions) error {
	owners, err := ver0_16_2ListOwners(db)
	if err != nil {
		return err
	}

	if len(owners) == 0 {
		return nil
	}

	seeds, err := ver0_16_2StandardBTFCatalogSeeds()
	if err != nil {
		return err
	}

	for _, owner := range owners {
		for i := range seeds {
			insertErr := ver0_16_2InsertStandardBTFCatalogSeed(db, owner, &seeds[i])
			if insertErr != nil {
				return insertErr
			}
		}
	}

	return nil
}

func ver0_16_2SeedEbicsStandardBTFCatalogsDown(db Actions) error {
	if err := db.Exec(
		`DELETE FROM ebics_standard_btf_catalogs
		  WHERE name=? AND catalog_version=? AND source_type=?`,
		"gateway-standard-btf",
		"curated-country-pack-v1",
		"CUSTOM_OVERRIDE",
	); err != nil {
		return fmt.Errorf("failed to remove the seeded EBICS standard BTF catalogs: %w", err)
	}

	return nil
}

func ver0_16_2ListOwners(db Actions) ([]string, error) {
	rows, err := db.Query(`SELECT DISTINCT owner FROM users ORDER BY owner`)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve owners for the EBICS standard BTF seed: %w", err)
	}
	defer rows.Close()

	var owners []string
	for rows.Next() {
		var owner string
		if scanErr := rows.Scan(&owner); scanErr != nil {
			return nil, fmt.Errorf("failed to scan owner for the EBICS standard BTF seed: %w", scanErr)
		}
		owners = append(owners, owner)
	}

	rowsErr := rows.Err()
	if rowsErr != nil {
		return nil, fmt.Errorf("failed to iterate owners for the EBICS standard BTF seed: %w", rowsErr)
	}

	return owners, nil
}

func ver0_16_2InsertStandardBTFCatalogSeed(
	db Actions,
	owner string,
	seed *ver0_16_2StandardBTFCatalogSeed,
) error {
	checksum, err := ver0_16_2SeedChecksum(seed)
	if err != nil {
		return fmt.Errorf("failed to compute the EBICS standard BTF seed checksum for scope %q: %w", seed.Scope, err)
	}

	execErr := db.Exec(
		`INSERT INTO ebics_standard_btf_catalogs(
			owner, name, scope, catalog_version, source_type, source_ref, status, seed_checksum
		) VALUES(?,?,?,?,?,?,?,?)`,
		owner,
		seed.Name,
		seed.Scope,
		seed.CatalogVersion,
		seed.SourceType,
		seed.SourceRef,
		seed.Status,
		checksum,
	)
	if execErr != nil {
		return fmt.Errorf(
			"failed to insert the EBICS standard BTF catalog seed for owner %q and scope %q: %w",
			owner,
			seed.Scope,
			execErr,
		)
	}

	var catalogID int64
	row := db.QueryRow(
		`SELECT id FROM ebics_standard_btf_catalogs
		  WHERE owner=? AND name=? AND scope=? AND catalog_version=?`,
		owner,
		seed.Name,
		seed.Scope,
		seed.CatalogVersion,
	)
	scanErr := row.Scan(&catalogID)
	if scanErr != nil {
		return fmt.Errorf(
			"failed to retrieve the inserted EBICS standard BTF catalog ID for owner %q and scope %q: %w",
			owner,
			seed.Scope,
			scanErr,
		)
	}

	for i := range seed.Entries {
		entry := &seed.Entries[i]
		metadata, marshalErr := json.Marshal(entry.Metadata)
		if marshalErr != nil {
			return fmt.Errorf(
				"failed to serialize the EBICS standard BTF seed metadata for entry %q: %w",
				entry.EntryKey,
				marshalErr,
			)
		}

		entryErr := db.Exec(
			`INSERT INTO ebics_standard_btf_entries(
				owner, catalog_id, entry_key, order_type, direction,
				service_name, service_option, scope, msg_name, container_type,
				country_group, is_default_template, status, metadata
			) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			owner,
			catalogID,
			entry.EntryKey,
			entry.OrderType,
			entry.Direction,
			entry.ServiceName,
			entry.ServiceOption,
			entry.Scope,
			entry.MsgName,
			entry.ContainerType,
			entry.CountryGroup,
			entry.IsDefaultTemplate,
			"ACTIVE",
			string(metadata),
		)
		if entryErr != nil {
			return fmt.Errorf(
				"failed to insert the EBICS standard BTF seed entry %q for owner %q and scope %q: %w",
				entry.EntryKey,
				owner,
				seed.Scope,
				entryErr,
			)
		}
	}

	return nil
}

func ver0_16_2SeedChecksum(seed *ver0_16_2StandardBTFCatalogSeed) (string, error) {
	payload, err := json.Marshal(seed)
	if err != nil {
		return "", fmt.Errorf("marshal EBICS standard BTF seed payload: %w", err)
	}

	sum := sha256.Sum256(payload)

	return hex.EncodeToString(sum[:]), nil
}

func ver0_16_2StandardBTFCatalogSeeds() ([]ver0_16_2StandardBTFCatalogSeed, error) {
	catalogs, err := ebicsbtfseed.DefaultCatalogs()
	if err != nil {
		return nil, fmt.Errorf("load canonical standard BTF catalogs: %w", err)
	}

	seeds := make([]ver0_16_2StandardBTFCatalogSeed, 0, len(catalogs))
	for i := range catalogs {
		catalog := &catalogs[i]
		seed := ver0_16_2StandardBTFCatalogSeed{
			Name:           catalog.Name,
			Scope:          catalog.Scope,
			CatalogVersion: catalog.CatalogVersion,
			SourceType:     catalog.SourceType,
			SourceRef:      catalog.SourceRef,
			Status:         catalog.Status,
			Entries:        make([]ver0_16_2StandardBTFEntrySeed, 0, len(catalog.Entries)),
		}

		for j := range catalog.Entries {
			entry := &catalog.Entries[j]
			seed.Entries = append(seed.Entries, ver0_16_2StandardBTFEntrySeed{
				EntryKey:          entry.EntryKey,
				OrderType:         entry.OrderType,
				Direction:         entry.Direction,
				ServiceName:       entry.ServiceName,
				ServiceOption:     entry.ServiceOption,
				Scope:             entry.Scope,
				MsgName:           entry.MsgName,
				ContainerType:     entry.ContainerType,
				CountryGroup:      entry.CountryGroup,
				IsDefaultTemplate: entry.IsDefaultTemplate,
				Metadata:          entry.Metadata,
			})
		}

		seeds = append(seeds, seed)
	}

	return seeds, nil
}
