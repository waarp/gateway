package model

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

const (
	ebicsStandardBTFCatalogScopeGLB = "GLB"
	ebicsStandardBTFCatalogScopeFR  = "FR"
	ebicsStandardBTFCatalogScopeDE  = "DE"
	ebicsStandardBTFCatalogScopeAT  = "AT"
	ebicsStandardBTFCatalogScopeCH  = "CH"

	ebicsStandardBTFCatalogSourceEmbedded = "LIB_EMBEDDED"
	ebicsStandardBTFCatalogSourceAnnex    = "OFFICIAL_ANNEX"
	ebicsStandardBTFCatalogSourceCustom   = "CUSTOM_OVERRIDE"

	ebicsStandardBTFCatalogStatusActive     = "ACTIVE"
	ebicsStandardBTFCatalogStatusSuperseded = "SUPERSEDED"
	ebicsStandardBTFCatalogStatusDisabled   = "DISABLED"
)

const (
	EbicsStandardBTFScopeGLB = ebicsStandardBTFCatalogScopeGLB
	EbicsStandardBTFScopeFR  = ebicsStandardBTFCatalogScopeFR
	EbicsStandardBTFScopeDE  = ebicsStandardBTFCatalogScopeDE
	EbicsStandardBTFScopeAT  = ebicsStandardBTFCatalogScopeAT
	EbicsStandardBTFScopeCH  = ebicsStandardBTFCatalogScopeCH
)

// EbicsStandardBTFCatalog stores one versioned standard BTF catalog for one scope.
type EbicsStandardBTFCatalog struct {
	ID    int64  `xorm:"<- id AUTOINCR"`
	Owner string `xorm:"owner"`

	Name           string `xorm:"name"`
	Scope          string `xorm:"scope"`
	CatalogVersion string `xorm:"catalog_version"`
	SourceType     string `xorm:"source_type"`
	SourceRef      string `xorm:"source_ref"`
	Status         string `xorm:"status"`
	SeedChecksum   string `xorm:"seed_checksum"`

	CreatedAt time.Time `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt time.Time `xorm:"updated_at UPDATED DATETIME(6) UTC"`
}

func (*EbicsStandardBTFCatalog) TableName() string   { return TableEbicsStandardBTFCatalogs }
func (*EbicsStandardBTFCatalog) Appellation() string { return NameEbicsStandardBTFCatalog }
func (c *EbicsStandardBTFCatalog) GetID() int64      { return c.ID }

func (c *EbicsStandardBTFCatalog) BeforeWrite(db database.Access) error {
	c.Owner = conf.GlobalConfig.GatewayName
	c.Name = strings.TrimSpace(c.Name)
	c.Scope = strings.ToUpper(strings.TrimSpace(c.Scope))
	c.CatalogVersion = strings.TrimSpace(c.CatalogVersion)
	c.SourceType = strings.ToUpper(strings.TrimSpace(c.SourceType))
	c.SourceRef = strings.TrimSpace(c.SourceRef)
	c.Status = strings.ToUpper(strings.TrimSpace(c.Status))
	c.SeedChecksum = strings.TrimSpace(c.SeedChecksum)

	if c.Name == "" {
		return database.NewValidationError("the EBICS standard BTF catalog name cannot be empty")
	}
	if err := validateEbicsStandardBTFCatalogScope(c.Scope); err != nil {
		return err
	}
	if c.CatalogVersion == "" {
		return database.NewValidationError("the EBICS standard BTF catalog version cannot be empty")
	}
	if err := validateEbicsStandardBTFCatalogSourceType(c.SourceType); err != nil {
		return err
	}
	if err := validateEbicsStandardBTFCatalogStatus(c.Status); err != nil {
		return err
	}

	if n, err := db.Count(c).Where(
		"id<>? AND owner=? AND name=? AND scope=? AND catalog_version=?",
		c.ID,
		c.Owner,
		c.Name,
		c.Scope,
		c.CatalogVersion,
	).Run(); err != nil {
		return fmt.Errorf("failed to check duplicate EBICS standard BTF catalogs: %w", err)
	} else if n != 0 {
		return database.NewValidationErrorf(
			"an EBICS standard BTF catalog %q/%q version %q already exists",
			c.Name,
			c.Scope,
			c.CatalogVersion,
		)
	}

	return nil
}

func validateEbicsStandardBTFCatalogScope(scope string) error {
	switch scope {
	case ebicsStandardBTFCatalogScopeGLB,
		ebicsStandardBTFCatalogScopeFR,
		ebicsStandardBTFCatalogScopeDE,
		ebicsStandardBTFCatalogScopeAT,
		ebicsStandardBTFCatalogScopeCH:
		return nil
	case "":
		return database.NewValidationError("the EBICS standard BTF catalog scope is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS standard BTF catalog scope", scope)
	}
}

func validateEbicsStandardBTFCatalogSourceType(sourceType string) error {
	switch sourceType {
	case ebicsStandardBTFCatalogSourceEmbedded,
		ebicsStandardBTFCatalogSourceAnnex,
		ebicsStandardBTFCatalogSourceCustom:
		return nil
	case "":
		return database.NewValidationError("the EBICS standard BTF catalog source type is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS standard BTF catalog source type", sourceType)
	}
}

func validateEbicsStandardBTFCatalogStatus(status string) error {
	switch status {
	case ebicsStandardBTFCatalogStatusActive,
		ebicsStandardBTFCatalogStatusSuperseded,
		ebicsStandardBTFCatalogStatusDisabled:
		return nil
	case "":
		return database.NewValidationError("the EBICS standard BTF catalog status is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS standard BTF catalog status", status)
	}
}

func (c *EbicsStandardBTFCatalog) Init(db database.Access) error {
	seeds := defaultEbicsStandardBTFCatalogSeeds()
	for i := range seeds {
		seed := &seeds[i]

		existing := &EbicsStandardBTFCatalog{}
		err := db.Get(
			existing,
			"owner=? AND name=? AND scope=? AND catalog_version=?",
			conf.GlobalConfig.GatewayName,
			seed.Name,
			seed.Scope,
			seed.CatalogVersion,
		).Run()
		if database.IsNotFound(err) {
			catalog := &EbicsStandardBTFCatalog{
				Name:           seed.Name,
				Scope:          seed.Scope,
				CatalogVersion: seed.CatalogVersion,
				SourceType:     seed.SourceType,
				SourceRef:      seed.SourceRef,
				Status:         seed.Status,
				SeedChecksum:   seedChecksumForCatalogSeed(seed),
			}
			if insertErr := db.Insert(catalog).Run(); insertErr != nil {
				return fmt.Errorf(
					"failed to insert the default EBICS standard BTF catalog %q/%q: %w",
					seed.Name,
					seed.Scope,
					insertErr,
				)
			}

			for j := range seed.Entries {
				entrySeed := &seed.Entries[j]
				entry := &EbicsStandardBTFEntry{
					CatalogID:         catalog.ID,
					EntryKey:          entrySeed.EntryKey,
					OrderType:         entrySeed.OrderType,
					Direction:         entrySeed.Direction,
					ServiceName:       entrySeed.ServiceName,
					ServiceOption:     entrySeed.ServiceOption,
					Scope:             entrySeed.Scope,
					MsgName:           entrySeed.MsgName,
					ContainerType:     entrySeed.ContainerType,
					CountryGroup:      entrySeed.CountryGroup,
					IsDefaultTemplate: entrySeed.IsDefaultTemplate,
					Status:            entrySeed.Status,
					MetadataMap:       entrySeed.Metadata,
				}
				if insertErr := db.Insert(entry).Run(); insertErr != nil {
					return fmt.Errorf(
						"failed to insert the default EBICS standard BTF entry %q for catalog %q/%q: %w",
						entrySeed.EntryKey,
						seed.Name,
						seed.Scope,
						insertErr,
					)
				}
			}

			continue
		}
		if err != nil {
			return fmt.Errorf(
				"failed to probe the default EBICS standard BTF catalog %q/%q: %w",
				seed.Name,
				seed.Scope,
				err,
			)
		}
	}

	return nil
}

type ebicsStandardBTFCatalogSeed struct {
	Name           string                      `json:"name"`
	Scope          string                      `json:"scope"`
	CatalogVersion string                      `json:"catalogVersion"`
	SourceType     string                      `json:"sourceType"`
	SourceRef      string                      `json:"sourceRef"`
	Status         string                      `json:"status"`
	Entries        []ebicsStandardBTFEntrySeed `json:"entries"`
}

type ebicsStandardBTFEntrySeed struct {
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
	Status            string         `json:"status"`
	Metadata          map[string]any `json:"metadata"`
}

func seedChecksumForCatalogSeed(seed *ebicsStandardBTFCatalogSeed) string {
	payload, err := json.Marshal(seed)
	if err != nil {
		return ""
	}

	sum := sha256.Sum256(payload)

	return hex.EncodeToString(sum[:])
}

func defaultEbicsStandardBTFCatalogSeeds() []ebicsStandardBTFCatalogSeed {
	const (
		catalogName       = "gateway-standard-btf"
		catalogVersion    = "2024-10-23-baseline-v1"
		catalogSourceType = "OFFICIAL_ANNEX"
		catalogSourceRef  = "EBICS Annex BTF ExternalCodeList 2024-10-23; Gateway curated baseline (non-exhaustive)"
	)

	commonMetadata := map[string]any{
		"seedOrigin":  "gateway-curated-baseline",
		"sourceSheet": "ServiceName",
		"exhaustive":  false,
	}

	return []ebicsStandardBTFCatalogSeed{
		{
			Name:           catalogName,
			Scope:          ebicsStandardBTFCatalogScopeGLB,
			CatalogVersion: catalogVersion,
			SourceType:     catalogSourceType,
			SourceRef:      catalogSourceRef,
			Status:         ebicsStandardBTFCatalogStatusActive,
			Entries: []ebicsStandardBTFEntrySeed{
				{
					EntryKey:          "btu-sct-pain001-xml",
					OrderType:         "BTU",
					Direction:         "UPLOAD",
					ServiceName:       "SCT",
					Scope:             "GLB",
					MsgName:           "pain.001",
					ContainerType:     "XML",
					IsDefaultTemplate: true,
					Status:            ebicsStandardBTFEntryStatusActive,
					Metadata:          standardBTFSeedMetadata(commonMetadata, "SEPA credit transfer baseline from annex examples"),
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
					Status:            ebicsStandardBTFEntryStatusActive,
					Metadata:          standardBTFSeedMetadata(commonMetadata, "SEPA direct debit baseline from annex examples"),
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
					Status:            ebicsStandardBTFEntryStatusActive,
					Metadata:          standardBTFSeedMetadata(commonMetadata, "Generic report baseline from annex examples"),
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
					Status:            ebicsStandardBTFEntryStatusActive,
					Metadata: standardBTFSeedMetadata(
						commonMetadata,
						"End-of-period statement baseline from annex examples",
					),
				},
				{
					EntryKey:          "btd-eop-mt940",
					OrderType:         "BTD",
					Direction:         "DOWNLOAD",
					ServiceName:       "EOP",
					Scope:             "GLB",
					MsgName:           "mt940",
					IsDefaultTemplate: true,
					Status:            ebicsStandardBTFEntryStatusActive,
					Metadata: standardBTFSeedMetadata(
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
					Status:            ebicsStandardBTFEntryStatusActive,
					Metadata:          standardBTFSeedMetadata(commonMetadata, "Intra-day statement baseline from annex examples"),
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
					Status:            ebicsStandardBTFEntryStatusActive,
					Metadata:          standardBTFSeedMetadata(commonMetadata, "Statement notification baseline from annex examples"),
				},
				{
					EntryKey:          "btd-stm-mt942",
					OrderType:         "BTD",
					Direction:         "DOWNLOAD",
					ServiceName:       "STM",
					Scope:             "GLB",
					MsgName:           "mt942",
					IsDefaultTemplate: true,
					Status:            ebicsStandardBTFEntryStatusActive,
					Metadata: standardBTFSeedMetadata(
						commonMetadata,
						"SWIFT intra-day statement baseline from annex examples",
					),
				},
			},
		},
		standardBTFCountrySeedCatalog(ebicsStandardBTFCatalogScopeFR, commonMetadata),
		standardBTFCountrySeedCatalog(ebicsStandardBTFCatalogScopeDE, commonMetadata),
		standardBTFCountrySeedCatalog(ebicsStandardBTFCatalogScopeAT, commonMetadata),
		standardBTFCountrySeedCatalog(ebicsStandardBTFCatalogScopeCH, commonMetadata),
	}
}

func standardBTFCountrySeedCatalog(scope string, commonMetadata map[string]any) ebicsStandardBTFCatalogSeed {
	const (
		name       = "gateway-standard-btf"
		version    = "2024-10-23-baseline-v1"
		sourceType = "OFFICIAL_ANNEX"
		sourceRef  = "EBICS Annex BTF ExternalCodeList 2024-10-23; " +
			"Gateway curated baseline (non-exhaustive)"
	)

	return ebicsStandardBTFCatalogSeed{
		Name:           name,
		Scope:          scope,
		CatalogVersion: version,
		SourceType:     sourceType,
		SourceRef:      sourceRef,
		Status:         ebicsStandardBTFCatalogStatusActive,
		Entries: []ebicsStandardBTFEntrySeed{
			{
				EntryKey:          "btu-dct-pain001-xml",
				OrderType:         "BTU",
				Direction:         "UPLOAD",
				ServiceName:       "DCT",
				Scope:             scope,
				MsgName:           "pain.001",
				ContainerType:     "XML",
				CountryGroup:      scope,
				IsDefaultTemplate: true,
				Status:            ebicsStandardBTFEntryStatusActive,
				Metadata:          standardBTFSeedMetadata(commonMetadata, "Domestic non-SEPA credit transfer baseline"),
			},
			{
				EntryKey:          "btu-ddd-pain008-xml",
				OrderType:         "BTU",
				Direction:         "UPLOAD",
				ServiceName:       "DDD",
				Scope:             scope,
				MsgName:           "pain.008",
				ContainerType:     "XML",
				CountryGroup:      scope,
				IsDefaultTemplate: true,
				Status:            ebicsStandardBTFEntryStatusActive,
				Metadata:          standardBTFSeedMetadata(commonMetadata, "Domestic non-SEPA direct debit baseline"),
			},
		},
	}
}

func standardBTFSeedMetadata(commonMetadata map[string]any, description string) map[string]any {
	metadata := make(map[string]any, len(commonMetadata)+1)
	maps.Copy(metadata, commonMetadata)
	metadata["description"] = description

	return metadata
}
