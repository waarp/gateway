package model

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/ebicsbtfseed"
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
	seeds, err := defaultEbicsStandardBTFCatalogSeeds()
	if err != nil {
		return fmt.Errorf("load the default EBICS standard BTF catalogs: %w", err)
	}

	for i := range seeds {
		seed := &seeds[i]

		existing := &EbicsStandardBTFCatalog{}
		getErr := db.Get(
			existing,
			"owner=? AND name=? AND scope=? AND catalog_version=?",
			conf.GlobalConfig.GatewayName,
			seed.Name,
			seed.Scope,
			seed.CatalogVersion,
		).Run()
		if database.IsNotFound(getErr) {
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
		if getErr != nil {
			return fmt.Errorf(
				"failed to probe the default EBICS standard BTF catalog %q/%q: %w",
				seed.Name,
				seed.Scope,
				getErr,
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

func defaultEbicsStandardBTFCatalogSeeds() ([]ebicsStandardBTFCatalogSeed, error) {
	catalogs, err := ebicsbtfseed.DefaultCatalogs()
	if err != nil {
		return nil, fmt.Errorf("load canonical standard BTF catalogs: %w", err)
	}

	seeds := make([]ebicsStandardBTFCatalogSeed, 0, len(catalogs))
	for i := range catalogs {
		catalog := &catalogs[i]
		seed := ebicsStandardBTFCatalogSeed{
			Name:           catalog.Name,
			Scope:          catalog.Scope,
			CatalogVersion: catalog.CatalogVersion,
			SourceType:     catalog.SourceType,
			SourceRef:      catalog.SourceRef,
			Status:         catalog.Status,
			Entries:        make([]ebicsStandardBTFEntrySeed, 0, len(catalog.Entries)),
		}

		for j := range catalog.Entries {
			entry := &catalog.Entries[j]
			seed.Entries = append(seed.Entries, ebicsStandardBTFEntrySeed{
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
				Status:            entry.Status,
				Metadata:          entry.Metadata,
			})
		}

		seeds = append(seeds, seed)
	}

	return seeds, nil
}
