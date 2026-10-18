package locale

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"slices"
	"strings"

	"code.waarp.fr/lib/gui"
	"github.com/goccy/go-yaml"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/webfs"
)

// CatalogFile is the translation catalogue of the web interface: a single
// YAML file, French and English side by side, keys nested by screen.
const CatalogFile = "i18n.yaml"

var (
	ErrCatalogEntry     = errors.New("a catalogue entry must map each language to a text")
	ErrCatalogDuplicate = errors.New("duplicate catalogue key")
)

// Catalog maps a dotted key ("services.title") to its texts by language.
type Catalog map[string]map[string]string

// Languages lists the languages every catalogue entry must provide.
func Languages() []string { return []string{"fr", "en"} }

// LoadCatalog reads CatalogFile from the web file system and merges the
// common strings of the GUI kit into it.
func LoadCatalog() (Catalog, error) {
	raw, err := fs.ReadFile(webfs.WebFS, CatalogFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read the catalogue %q: %w", CatalogFile, err)
	}

	catalog, err := ParseCatalog(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the catalogue %q: %w", CatalogFile, err)
	}

	common, err := ParseCatalog(gui.CommonCatalog)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the common catalogue of the kit: %w", err)
	}

	if mergeErr := catalog.Merge(common); mergeErr != nil {
		return nil, mergeErr
	}

	return catalog, nil
}

// MustLoadCatalog is LoadCatalog for the initialisation of a handler: the
// catalogue is embedded, failing to load it is a programming error.
func MustLoadCatalog() Catalog {
	catalog, err := LoadCatalog()
	if err != nil {
		panic(err)
	}

	return catalog
}

// ParseCatalog reads a YAML catalogue. Sections nest freely; an entry maps
// each language code to a text: `title: { fr: État des services, en: Service status }`.
func ParseCatalog(src []byte) (Catalog, error) {
	var tree map[string]any
	if err := yaml.Unmarshal(src, &tree); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}

	catalog := Catalog{}
	if err := flatten(catalog, "", tree); err != nil {
		return nil, err
	}

	return catalog, nil
}

func flatten(catalog Catalog, prefix string, node map[string]any) error {
	for key, value := range node {
		path := dotted(prefix, key)

		child, isMap := value.(map[string]any)
		if !isMap {
			return fmt.Errorf("%w: %q", ErrCatalogEntry, path)
		}

		if !hasLanguageKey(child) {
			if err := flatten(catalog, path, child); err != nil {
				return err
			}

			continue
		}

		texts, isEntry := entryTexts(child)
		if !isEntry {
			return fmt.Errorf("%w: %q", ErrCatalogEntry, path)
		}

		catalog[path] = texts
	}

	return nil
}

// dotted joins a section path and a key ("services" + "title" gives
// "services.title").
func dotted(prefix, key string) string {
	if prefix == "" {
		return key
	}

	return prefix + "." + key
}

// hasLanguageKey tells a {language: text} entry from a section: one of its
// keys is a language code.
func hasLanguageKey(node map[string]any) bool {
	for key := range node {
		if slices.Contains(Languages(), key) {
			return true
		}
	}

	return false
}

// entryTexts returns the texts of an entry; an entry maps language codes,
// and nothing else, to texts.
func entryTexts(node map[string]any) (map[string]string, bool) {
	texts := make(map[string]string, len(node))

	for lang, value := range node {
		text, isText := value.(string)
		if !isText || !slices.Contains(Languages(), lang) {
			return nil, false
		}

		texts[lang] = text
	}

	return texts, true
}

// Merge adds the entries of other to the catalogue. A key present on both
// sides is an error, so that every text has a single source.
func (c Catalog) Merge(other Catalog) error {
	for key, texts := range other {
		if _, duplicate := c[key]; duplicate {
			return fmt.Errorf("%w: %q", ErrCatalogDuplicate, key)
		}

		c[key] = texts
	}

	return nil
}

// Has reports whether the catalogue has a text for the key in every language.
func (c Catalog) Has(key string) bool {
	for _, lang := range Languages() {
		if c[key][lang] == "" {
			return false
		}
	}

	return true
}

// Keys returns the keys of the catalogue, sorted.
func (c Catalog) Keys() []string {
	return slices.Sorted(maps.Keys(c))
}

// Translator binds the catalogue to a language, for gui.Localize. Parameters
// are given as alternating name/value pairs and replace their {name}
// placeholder. A missing text shows as "¿key?" instead of failing the page;
// the catalogue test keeps it from reaching a release.
func (c Catalog) Translator(lang string) gui.Translator {
	return func(key string, args ...any) string {
		text := c[key][lang]
		if text == "" {
			return "¿" + key + "?"
		}

		for i := 0; i+1 < len(args); i += 2 {
			text = strings.ReplaceAll(text, "{"+fmt.Sprint(args[i])+"}", fmt.Sprint(args[i+1]))
		}

		return text
	}
}
