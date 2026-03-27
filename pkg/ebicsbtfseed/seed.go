package ebicsbtfseed

import (
	_ "embed"
	"encoding/json"
	"fmt"

	backupfile "code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
)

//go:embed default_catalogs.json
var defaultCatalogsJSON []byte

// DefaultCatalogs returns the canonical standard BTF catalogs embedded in Gateway.
func DefaultCatalogs() ([]backupfile.EbicsStandardBTFCatalog, error) {
	data := &backupfile.Data{}
	if err := json.Unmarshal(defaultCatalogsJSON, data); err != nil {
		return nil, fmt.Errorf("decode embedded EBICS standard BTF catalogs: %w", err)
	}

	return data.EbicsStandardBTFCatalogs, nil
}
