package migrations

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
)

const (
	ver0_16_2StandardBTFCatalogName       = "gateway-standard-btf"
	ver0_16_2StandardBTFCatalogVersion    = "2024-10-23-baseline-v1"
	ver0_16_2StandardBTFCatalogSourceType = "OFFICIAL_ANNEX"
	ver0_16_2StandardBTFCatalogSourceRef  = "EBICS Annex BTF ExternalCodeList 2024-10-23; " +
		"Gateway curated baseline (non-exhaustive)"
)

type ver0_16_2StandardBTFCatalogSeed struct {
	Scope   string                          `json:"scope"`
	Entries []ver0_16_2StandardBTFEntrySeed `json:"entries"`
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

	seeds := ver0_16_2StandardBTFCatalogSeeds()
	for _, owner := range owners {
		for _, seed := range seeds {
			insertErr := ver0_16_2InsertStandardBTFCatalogSeed(db, owner, seed)
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
		ver0_16_2StandardBTFCatalogName,
		ver0_16_2StandardBTFCatalogVersion,
		ver0_16_2StandardBTFCatalogSourceType,
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
	seed ver0_16_2StandardBTFCatalogSeed,
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
		ver0_16_2StandardBTFCatalogName,
		seed.Scope,
		ver0_16_2StandardBTFCatalogVersion,
		ver0_16_2StandardBTFCatalogSourceType,
		ver0_16_2StandardBTFCatalogSourceRef,
		"ACTIVE",
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
		ver0_16_2StandardBTFCatalogName,
		seed.Scope,
		ver0_16_2StandardBTFCatalogVersion,
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

func ver0_16_2SeedChecksum(seed ver0_16_2StandardBTFCatalogSeed) (string, error) {
	payload, err := json.Marshal(seed)
	if err != nil {
		return "", fmt.Errorf("marshal EBICS standard BTF seed payload: %w", err)
	}

	sum := sha256.Sum256(payload)

	return hex.EncodeToString(sum[:]), nil
}

func ver0_16_2StandardBTFCatalogSeeds() []ver0_16_2StandardBTFCatalogSeed {
	commonMetadata := map[string]any{
		"seedOrigin":  "gateway-curated-baseline",
		"sourceSheet": "ServiceName",
		"exhaustive":  false,
	}

	return []ver0_16_2StandardBTFCatalogSeed{
		{
			Scope: "GLB",
			Entries: []ver0_16_2StandardBTFEntrySeed{
				{
					EntryKey:          "btu-sct-pain001-xml",
					OrderType:         "BTU",
					Direction:         "UPLOAD",
					ServiceName:       "SCT",
					Scope:             "GLB",
					MsgName:           "pain.001",
					ContainerType:     "XML",
					IsDefaultTemplate: true,
					Metadata:          ver0_16_2EntryMetadata(commonMetadata, "SEPA credit transfer baseline from annex examples"),
				},
				{
					EntryKey:          "btu-sdd-pain008-xml",
					OrderType:         "BTU",
					Direction:         "UPLOAD",
					ServiceName:       "SDD",
					Scope:             "GLB",
					MsgName:           "pain.008",
					ContainerType:     "XML",
					IsDefaultTemplate: true,
					Metadata:          ver0_16_2EntryMetadata(commonMetadata, "SEPA direct debit baseline from annex examples"),
				},
				{
					EntryKey:          "btd-rep-pain002-xml",
					OrderType:         "BTD",
					Direction:         "DOWNLOAD",
					ServiceName:       "REP",
					Scope:             "GLB",
					MsgName:           "pain.002",
					ContainerType:     "XML",
					IsDefaultTemplate: true,
					Metadata:          ver0_16_2EntryMetadata(commonMetadata, "Generic report baseline from annex examples"),
				},
				{
					EntryKey:          "btd-eop-camt053-xml",
					OrderType:         "BTD",
					Direction:         "DOWNLOAD",
					ServiceName:       "EOP",
					Scope:             "GLB",
					MsgName:           "camt.053",
					ContainerType:     "XML",
					IsDefaultTemplate: true,
					Metadata:          ver0_16_2EntryMetadata(commonMetadata, "End-of-period statement baseline from annex examples"),
				},
				{
					EntryKey:          "btd-eop-mt940",
					OrderType:         "BTD",
					Direction:         "DOWNLOAD",
					ServiceName:       "EOP",
					Scope:             "GLB",
					MsgName:           "mt940",
					IsDefaultTemplate: true,
					Metadata: ver0_16_2EntryMetadata(
						commonMetadata,
						"SWIFT end-of-period statement baseline from annex examples",
					),
				},
				{
					EntryKey:          "btd-stm-camt052-xml",
					OrderType:         "BTD",
					Direction:         "DOWNLOAD",
					ServiceName:       "STM",
					Scope:             "GLB",
					MsgName:           "camt.052",
					ContainerType:     "XML",
					IsDefaultTemplate: true,
					Metadata:          ver0_16_2EntryMetadata(commonMetadata, "Intra-day statement baseline from annex examples"),
				},
				{
					EntryKey:          "btd-stm-camt054-xml",
					OrderType:         "BTD",
					Direction:         "DOWNLOAD",
					ServiceName:       "STM",
					Scope:             "GLB",
					MsgName:           "camt.054",
					ContainerType:     "XML",
					IsDefaultTemplate: true,
					Metadata:          ver0_16_2EntryMetadata(commonMetadata, "Statement notification baseline from annex examples"),
				},
				{
					EntryKey:          "btd-stm-mt942",
					OrderType:         "BTD",
					Direction:         "DOWNLOAD",
					ServiceName:       "STM",
					Scope:             "GLB",
					MsgName:           "mt942",
					IsDefaultTemplate: true,
					Metadata: ver0_16_2EntryMetadata(
						commonMetadata,
						"SWIFT intra-day statement baseline from annex examples",
					),
				},
			},
		},
		{
			Scope: "FR",
			Entries: []ver0_16_2StandardBTFEntrySeed{
				{
					EntryKey:          "btu-dct-pain001-xml",
					OrderType:         "BTU",
					Direction:         "UPLOAD",
					ServiceName:       "DCT",
					Scope:             "FR",
					MsgName:           "pain.001",
					ContainerType:     "XML",
					CountryGroup:      "FR",
					IsDefaultTemplate: true,
					Metadata:          ver0_16_2EntryMetadata(commonMetadata, "Domestic non-SEPA credit transfer baseline"),
				},
				{
					EntryKey:          "btu-ddd-pain008-xml",
					OrderType:         "BTU",
					Direction:         "UPLOAD",
					ServiceName:       "DDD",
					Scope:             "FR",
					MsgName:           "pain.008",
					ContainerType:     "XML",
					CountryGroup:      "FR",
					IsDefaultTemplate: true,
					Metadata:          ver0_16_2EntryMetadata(commonMetadata, "Domestic non-SEPA direct debit baseline"),
				},
			},
		},
		{
			Scope: "DE",
			Entries: []ver0_16_2StandardBTFEntrySeed{
				{
					EntryKey:          "btu-dct-pain001-xml",
					OrderType:         "BTU",
					Direction:         "UPLOAD",
					ServiceName:       "DCT",
					Scope:             "DE",
					MsgName:           "pain.001",
					ContainerType:     "XML",
					CountryGroup:      "DE",
					IsDefaultTemplate: true,
					Metadata:          ver0_16_2EntryMetadata(commonMetadata, "Domestic non-SEPA credit transfer baseline"),
				},
				{
					EntryKey:          "btu-ddd-pain008-xml",
					OrderType:         "BTU",
					Direction:         "UPLOAD",
					ServiceName:       "DDD",
					Scope:             "DE",
					MsgName:           "pain.008",
					ContainerType:     "XML",
					CountryGroup:      "DE",
					IsDefaultTemplate: true,
					Metadata:          ver0_16_2EntryMetadata(commonMetadata, "Domestic non-SEPA direct debit baseline"),
				},
			},
		},
		{
			Scope: "AT",
			Entries: []ver0_16_2StandardBTFEntrySeed{
				{
					EntryKey:          "btu-dct-pain001-xml",
					OrderType:         "BTU",
					Direction:         "UPLOAD",
					ServiceName:       "DCT",
					Scope:             "AT",
					MsgName:           "pain.001",
					ContainerType:     "XML",
					CountryGroup:      "AT",
					IsDefaultTemplate: true,
					Metadata:          ver0_16_2EntryMetadata(commonMetadata, "Domestic non-SEPA credit transfer baseline"),
				},
				{
					EntryKey:          "btu-ddd-pain008-xml",
					OrderType:         "BTU",
					Direction:         "UPLOAD",
					ServiceName:       "DDD",
					Scope:             "AT",
					MsgName:           "pain.008",
					ContainerType:     "XML",
					CountryGroup:      "AT",
					IsDefaultTemplate: true,
					Metadata:          ver0_16_2EntryMetadata(commonMetadata, "Domestic non-SEPA direct debit baseline"),
				},
			},
		},
		{
			Scope: "CH",
			Entries: []ver0_16_2StandardBTFEntrySeed{
				{
					EntryKey:          "btu-dct-pain001-xml",
					OrderType:         "BTU",
					Direction:         "UPLOAD",
					ServiceName:       "DCT",
					Scope:             "CH",
					MsgName:           "pain.001",
					ContainerType:     "XML",
					CountryGroup:      "CH",
					IsDefaultTemplate: true,
					Metadata:          ver0_16_2EntryMetadata(commonMetadata, "Domestic non-SEPA credit transfer baseline"),
				},
				{
					EntryKey:          "btu-ddd-pain008-xml",
					OrderType:         "BTU",
					Direction:         "UPLOAD",
					ServiceName:       "DDD",
					Scope:             "CH",
					MsgName:           "pain.008",
					ContainerType:     "XML",
					CountryGroup:      "CH",
					IsDefaultTemplate: true,
					Metadata:          ver0_16_2EntryMetadata(commonMetadata, "Domestic non-SEPA direct debit baseline"),
				},
			},
		},
	}
}

func ver0_16_2EntryMetadata(base map[string]any, description string) map[string]any {
	out := make(map[string]any, len(base)+1)
	maps.Copy(out, base)
	out["description"] = description

	return out
}
