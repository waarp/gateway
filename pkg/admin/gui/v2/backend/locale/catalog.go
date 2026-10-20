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

// CatalogDir holds the translation catalogue of the web interface: one YAML
// file per language (fr.yaml, en.yaml), the same key tree in every file,
// keys nested by screen, texts as leaves.
const CatalogDir = "i18n"

var (
	ErrCatalogEntry     = errors.New("a catalogue entry must be a text")
	ErrCatalogDuplicate = errors.New("duplicate catalogue key")
)

// Catalog maps a dotted key ("services.title") to its texts by language.
type Catalog map[string]map[string]string

// Languages lists the languages of the interface, those the kit ships.
func Languages() []string { return slices.Clone(gui.Languages) }

// LoadCatalog reads the file of every language from CatalogDir and merges the
// common strings of the kit into it.
func LoadCatalog() (Catalog, error) {
	catalog := Catalog{}

	for _, lang := range Languages() {
		name := lang + ".yaml"

		if err := catalog.load(lang, webfs.WebFS, CatalogDir+"/"+name); err != nil {
			return nil, err
		}

		if err := catalog.load(lang, gui.I18n, name); err != nil {
			return nil, fmt.Errorf("kit: %w", err)
		}
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

// load adds the texts of one language file to the catalogue. A key already
// holding a text in that language is an error, so that every text has a
// single source.
func (c Catalog) load(lang string, fsys fs.FS, name string) error {
	raw, err := fs.ReadFile(fsys, name)
	if err != nil {
		return fmt.Errorf("failed to read the catalogue %q: %w", name, err)
	}

	texts, err := ParseLanguage(raw)
	if err != nil {
		return fmt.Errorf("failed to parse the catalogue %q: %w", name, err)
	}

	return c.Add(lang, texts)
}

// ParseLanguage reads the YAML catalogue of one language. Sections nest
// freely; a leaf is a text: `services:\n  title: État des services`.
func ParseLanguage(src []byte) (map[string]string, error) {
	var tree map[string]any
	if err := yaml.Unmarshal(src, &tree); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}

	texts := map[string]string{}
	if err := flatten(texts, "", tree); err != nil {
		return nil, err
	}

	return texts, nil
}

func flatten(texts map[string]string, prefix string, node map[string]any) error {
	for key, value := range node {
		path := dotted(prefix, key)

		switch child := value.(type) {
		case string:
			texts[path] = child
		case map[string]any:
			if err := flatten(texts, path, child); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%w: %q", ErrCatalogEntry, path)
		}
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

// Add puts the texts of one language into the catalogue; a key that already
// has a text in that language is an error.
func (c Catalog) Add(lang string, texts map[string]string) error {
	for key, text := range texts {
		if c[key][lang] != "" {
			return fmt.Errorf("%w: %q (%s)", ErrCatalogDuplicate, key, lang)
		}

		if c[key] == nil {
			c[key] = map[string]string{}
		}

		c[key][lang] = text
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
