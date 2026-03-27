package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	backupfile "code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
)

const (
	catalogName        = "gateway-standard-btf"
	catalogVersion     = "curated-country-pack-v1"
	catalogSourceType  = "CUSTOM_OVERRIDE"
	catalogStatus      = "ACTIVE"
	defaultInputPath   = `C:\MonProjet\Waarp_Ebics\assets\btf\curated`
	defaultOutputDir   = `pkg\backup\testdata`
	defaultJSONName    = "ebics-standard-btf-curated.json"
	defaultYAMLName    = "ebics-standard-btf-curated.yaml"
	globalCatalogScope = "GLB"
	sourceRefPrefix    = "Waarp_Ebics/assets/btf/curated"
	minCSVRecordCount  = 2
	secondArgIndex     = 2
	dirPermissions     = 0o755
	filePermissions    = 0o644
	orderTypeBTU       = "BTU"
	orderTypeBTD       = "BTD"
	directionUpload    = "UPLOAD"
	directionDownload  = "DOWNLOAD"
)

var (
	errCuratedCSVEmpty    = errors.New("curated CSV is empty")
	errMissingOrderKind   = errors.New("missing protocol order kind")
	errUnsupportedDirKind = errors.New("unsupported transfer direction")
)

type curatedRow struct {
	SourceID           string
	Country            string
	FlowID             string
	ServiceScope       string
	ServiceName        string
	ServiceOption      string
	ServiceMsgName     string
	BTFOrderType       string
	Variant            string
	ProtocolOrderKind  string
	TransferDirection  string
	BusinessDirection  string
	ServiceNameLabel   string
	ServiceOptionLabel string
	EvidenceRef        string
}

func main() {
	inputDir := defaultInputPath
	outputDir := defaultOutputDir
	if len(os.Args) > 1 {
		inputDir = os.Args[1]
	}
	if len(os.Args) > secondArgIndex {
		outputDir = os.Args[2]
	}

	data, err := buildImportData(inputDir)
	if err != nil {
		fatalf("failed to build the curated EBICS standard BTF import data: %v", err)
	}

	if err = os.MkdirAll(outputDir, dirPermissions); err != nil {
		fatalf("failed to create output directory %q: %v", outputDir, err)
	}

	jsonPath := filepath.Join(outputDir, defaultJSONName)
	yamlPath := filepath.Join(outputDir, defaultYAMLName)

	if err = writeJSON(jsonPath, data); err != nil {
		fatalf("failed to write %q: %v", jsonPath, err)
	}
	if err = writeYAML(yamlPath, data); err != nil {
		fatalf("failed to write %q: %v", yamlPath, err)
	}

	fmt.Printf("Generated %s and %s\n", jsonPath, yamlPath)
}

func buildImportData(inputDir string) (*backupfile.Data, error) {
	entriesByCatalog := map[string][]backupfile.EbicsStandardBTFEntry{}
	glbSeen := map[string]int{}

	for _, country := range []string{"fr", "de", "at", "ch"} {
		rows, err := loadCuratedRows(filepath.Join(inputDir, country+".csv"))
		if err != nil {
			return nil, err
		}

		countryScope := strings.ToUpper(country)
		countryKey := catalogKey(countryScope)
		for i := range rows {
			row := &rows[i]
			entry, convErr := curatedRowToEntry(row)
			if convErr != nil {
				return nil, fmt.Errorf("convert curated row for country %q: %w", countryScope, convErr)
			}

			entriesByCatalog[countryKey] = append(entriesByCatalog[countryKey], entry)

			if entry.Scope != globalCatalogScope {
				continue
			}

			glbKey := dedupeKey(&entry)
			if idx, ok := glbSeen[glbKey]; ok {
				mergeGlobalMetadata(&entriesByCatalog[catalogKey(globalCatalogScope)][idx], row)
				continue
			}

			glbEntry := cloneEntry(&entry)
			entriesByCatalog[catalogKey(globalCatalogScope)] = append(entriesByCatalog[catalogKey(globalCatalogScope)], glbEntry)
			glbSeen[glbKey] = len(entriesByCatalog[catalogKey(globalCatalogScope)]) - 1
		}
	}

	catalogScopes := []string{globalCatalogScope, "FR", "DE", "AT", "CH"}
	catalogs := make([]backupfile.EbicsStandardBTFCatalog, 0, len(catalogScopes))
	for _, scope := range catalogScopes {
		entries := entriesByCatalog[catalogKey(scope)]
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].EntryKey < entries[j].EntryKey
		})

		catalogs = append(catalogs, backupfile.EbicsStandardBTFCatalog{
			Name:           catalogName,
			Scope:          scope,
			CatalogVersion: catalogVersion,
			SourceType:     catalogSourceType,
			SourceRef:      fmt.Sprintf("%s (%s scope pack)", sourceRefPrefix, scope),
			Status:         catalogStatus,
			Entries:        entries,
		})
	}

	return &backupfile.Data{
		EbicsStandardBTFCatalogs: catalogs,
	}, nil
}

func loadCuratedRows(path string) ([]curatedRow, error) {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("open curated CSV %q: %w", path, err)
	}
	defer file.Close() //nolint:errcheck // best effort cleanup

	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read curated CSV %q: %w", path, err)
	}
	if len(records) < minCSVRecordCount {
		return nil, fmt.Errorf("%w: %q", errCuratedCSVEmpty, path)
	}

	indexByName := map[string]int{}
	for idx, name := range records[0] {
		indexByName[strings.TrimSpace(name)] = idx
	}

	rows := make([]curatedRow, 0, len(records)-1)
	for _, record := range records[1:] {
		row := curatedRow{
			SourceID:           csvValue(record, indexByName, "source_id"),
			Country:            strings.ToUpper(csvValue(record, indexByName, "country")),
			FlowID:             csvValue(record, indexByName, "flow_id"),
			ServiceScope:       strings.ToUpper(csvValue(record, indexByName, "service_scope")),
			ServiceName:        csvValue(record, indexByName, "service_name"),
			ServiceOption:      csvValue(record, indexByName, "service_option"),
			ServiceMsgName:     csvValue(record, indexByName, "service_msg_name"),
			BTFOrderType:       csvValueAlias(record, indexByName, "btf_order_type", "order_type"),
			Variant:            csvValue(record, indexByName, "variant"),
			ProtocolOrderKind:  strings.ToUpper(csvValue(record, indexByName, "protocol_order_kind")),
			TransferDirection:  strings.ToLower(csvValue(record, indexByName, "transfer_direction")),
			BusinessDirection:  csvValue(record, indexByName, "business_direction"),
			ServiceNameLabel:   csvValue(record, indexByName, "service_name_label"),
			ServiceOptionLabel: csvValue(record, indexByName, "service_option_label"),
			EvidenceRef:        csvValue(record, indexByName, "evidence_ref"),
		}

		if strings.TrimSpace(row.ServiceName) == "" || strings.TrimSpace(row.BTFOrderType) == "" {
			continue
		}
		rows = append(rows, row)
	}

	return rows, nil
}

func curatedRowToEntry(row *curatedRow) (backupfile.EbicsStandardBTFEntry, error) {
	orderType := normalizeOrderType(row.ProtocolOrderKind, row.BTFOrderType)
	if orderType == "" {
		return backupfile.EbicsStandardBTFEntry{}, errMissingOrderKind
	}

	direction := normalizeDirection(row.TransferDirection, orderType)
	if direction == "" {
		return backupfile.EbicsStandardBTFEntry{}, fmt.Errorf(
			"%w %q",
			errUnsupportedDirKind,
			row.TransferDirection,
		)
	}

	scope := row.ServiceScope
	if scope == "" {
		scope = globalCatalogScope
	}

	entry := backupfile.EbicsStandardBTFEntry{
		EntryKey: buildEntryKey(
			orderType,
			row.ServiceName,
			row.ServiceOption,
			scope,
			row.ServiceMsgName,
			inferredContainerType(row.ServiceMsgName),
			row.Variant,
		),
		OrderType:         orderType,
		Direction:         direction,
		ServiceName:       strings.TrimSpace(row.ServiceName),
		ServiceOption:     strings.TrimSpace(row.ServiceOption),
		Scope:             scope,
		MsgName:           strings.TrimSpace(row.ServiceMsgName),
		ContainerType:     inferredContainerType(row.ServiceMsgName),
		CountryGroup:      row.Country,
		IsDefaultTemplate: strings.TrimSpace(row.ServiceOption) == "",
		Status:            "ACTIVE",
		Metadata: map[string]any{
			"sourceId":          row.SourceID,
			"country":           row.Country,
			"flowID":            row.FlowID,
			"variant":           normalizeOptional(row.Variant),
			"businessDirection": normalizeOptional(row.BusinessDirection),
			"serviceNameLabel":  normalizeOptional(row.ServiceNameLabel),
			"serviceOptionLabel": normalizeOptional(
				row.ServiceOptionLabel,
			),
			"evidenceRef": normalizeOptional(row.EvidenceRef),
			"catalogKind": "curated-country-pack",
		},
	}

	return entry, nil
}

func mergeGlobalMetadata(entry *backupfile.EbicsStandardBTFEntry, row *curatedRow) {
	if entry.Metadata == nil {
		entry.Metadata = map[string]any{}
	}

	sourceCountries, _ := entry.Metadata["sourceCountries"].([]any)
	country := row.Country
	for _, value := range sourceCountries {
		if strings.EqualFold(fmt.Sprint(value), country) {
			return
		}
	}
	entry.Metadata["sourceCountries"] = append(sourceCountries, country)
}

func cloneEntry(entry *backupfile.EbicsStandardBTFEntry) backupfile.EbicsStandardBTFEntry {
	cloned := *entry
	cloned.CountryGroup = ""
	cloned.Metadata = cloneMetadata(entry.Metadata)
	if cloned.Metadata == nil {
		cloned.Metadata = map[string]any{}
	}
	cloned.Metadata["sourceCountries"] = []any{entry.CountryGroup}

	return cloned
}

func cloneMetadata(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}

	dst := make(map[string]any, len(src))
	maps.Copy(dst, src)

	return dst
}

func dedupeKey(entry *backupfile.EbicsStandardBTFEntry) string {
	return strings.Join([]string{
		entry.OrderType,
		entry.Direction,
		entry.ServiceName,
		entry.ServiceOption,
		entry.Scope,
		entry.MsgName,
		entry.ContainerType,
	}, "|")
}

func catalogKey(scope string) string {
	return strings.ToUpper(strings.TrimSpace(scope))
}

func normalizeOrderType(protocolOrderKind, btfOrderType string) string {
	switch strings.ToUpper(strings.TrimSpace(protocolOrderKind)) {
	case orderTypeBTU, "FUL":
		return orderTypeBTU
	case orderTypeBTD, "FDL":
		return orderTypeBTD
	}

	switch strings.ToUpper(strings.TrimSpace(btfOrderType)) {
	case orderTypeBTU, "FUL":
		return orderTypeBTU
	case orderTypeBTD, "FDL":
		return orderTypeBTD
	}

	return ""
}

func normalizeDirection(direction, orderType string) string {
	switch strings.ToLower(strings.TrimSpace(direction)) {
	case "send":
		return directionUpload
	case "receive":
		return directionDownload
	case "both":
		if orderType == orderTypeBTD {
			return directionDownload
		}

		return directionUpload
	case "":
		if orderType == orderTypeBTD {
			return directionDownload
		}

		return directionUpload
	default:
		return ""
	}
}

func buildEntryKey(orderType, serviceName, serviceOption, scope, msgName, containerType, variant string) string {
	parts := []string{
		strings.ToLower(strings.TrimSpace(orderType)),
		slugify(serviceName),
		slugify(serviceOption),
		slugify(scope),
		slugify(msgName),
		slugify(containerType),
		slugify(variant),
	}

	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" && part != "n-a" {
			filtered = append(filtered, part)
		}
	}

	return strings.Join(filtered, "-")
}

func inferredContainerType(msgName string) string {
	normalized := strings.ToLower(strings.TrimSpace(msgName))
	switch {
	case strings.HasPrefix(normalized, "pain."),
		strings.HasPrefix(normalized, "camt."),
		strings.HasPrefix(normalized, "acmt."):
		return "XML"
	default:
		return ""
	}
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}

	replacer := strings.NewReplacer(
		".", "",
		"_", "-",
		"/", "-",
		" ", "-",
		`"`, "",
		"'", "",
	)
	value = replacer.Replace(value)

	for strings.Contains(value, "--") {
		value = strings.ReplaceAll(value, "--", "-")
	}

	return strings.Trim(value, "-")
}

func normalizeOptional(value string) string {
	value = strings.TrimSpace(value)
	switch strings.ToLower(value) {
	case "", "n/a", "na", "none":
		return ""
	default:
		return value
	}
}

func csvValue(record []string, indexByName map[string]int, name string) string {
	idx, ok := indexByName[name]
	if !ok || idx >= len(record) {
		return ""
	}

	return strings.TrimSpace(record[idx])
}

func csvValueAlias(record []string, indexByName map[string]int, names ...string) string {
	for _, name := range names {
		if value := csvValue(record, indexByName, name); value != "" {
			return value
		}
	}

	return ""
}

func writeJSON(path string, data *backupfile.Data) error {
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}

	if err = os.WriteFile(path, append(content, '\n'), filePermissions); err != nil {
		return fmt.Errorf("write JSON file %q: %w", path, err)
	}

	return nil
}

func writeYAML(path string, data *backupfile.Data) error {
	content, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal YAML: %w", err)
	}

	if err = os.WriteFile(path, content, filePermissions); err != nil {
		return fmt.Errorf("write YAML file %q: %w", path, err)
	}

	return nil
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
