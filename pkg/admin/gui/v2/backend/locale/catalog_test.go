package locale_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"code.waarp.fr/lib/gui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/locale"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/webfs"
)

// Every key of the catalogue, the kit's common strings included, has a text
// in every language.
func TestCatalogIsComplete(t *testing.T) {
	t.Parallel()

	catalog, err := locale.LoadCatalog()
	require.NoError(t, err)
	require.NotEmpty(t, catalog)

	for _, key := range catalog.Keys() {
		for _, lang := range locale.Languages() {
			assert.NotEmptyf(t, catalog[key][lang], "key %q has no %s text", key, lang)
		}
	}
}

// Every key called by a template exists in the catalogue, and every key of the
// product catalogue is used by a template or a handler: no orphan.
func TestCatalogMatchesTemplates(t *testing.T) {
	t.Parallel()

	catalog, err := locale.LoadCatalog()
	require.NoError(t, err)

	raw, err := fs.ReadFile(webfs.WebFS, locale.CatalogFile)
	require.NoError(t, err)

	product, err := locale.ParseCatalog(raw)
	require.NoError(t, err)

	templates := readAll(t, webfs.WebFS, "*.gohtml", "*/*.gohtml", "*/*/*.gohtml")
	templates = append(templates, readAll(t, gui.Templates, "*/*.gohtml")...)
	sources := readGoSources(t, "..")

	call := regexp.MustCompile(`\bt "([A-Za-z0-9_.]+)"`)
	for name, content := range templates {
		for _, match := range call.FindAllStringSubmatch(content, -1) {
			assert.Truef(t, catalog.Has(match[1]), "%s calls key %q, missing from the catalogue", name, match[1])
		}
	}

	literal := regexp.MustCompile(`"([A-Za-z0-9_.]+\.?)"`)
	used := map[string]bool{}
	prefixes := map[string]bool{}

	for _, content := range append(templates, sources...) {
		for _, match := range literal.FindAllStringSubmatch(content, -1) {
			if strings.HasSuffix(match[1], ".") {
				prefixes[match[1]] = true
			} else {
				used[match[1]] = true
			}
		}
	}

	for _, key := range product.Keys() {
		assert.Truef(t, used[key] || hasPrefix(key, prefixes),
			"key %q is used by no template nor handler", key)
	}
}

func hasPrefix(key string, prefixes map[string]bool) bool {
	for prefix := range prefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}

	return false
}

func readAll(t *testing.T, fsys fs.FS, patterns ...string) []string {
	t.Helper()

	var contents []string

	for _, pattern := range patterns {
		names, err := fs.Glob(fsys, pattern)
		require.NoError(t, err)

		for _, name := range names {
			data, err := fs.ReadFile(fsys, name)
			require.NoError(t, err)

			contents = append(contents, string(data))
		}
	}

	return contents
}

func readGoSources(t *testing.T, root string) []string {
	t.Helper()

	var contents []string

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		contents = append(contents, string(data))

		return nil
	})
	require.NoError(t, err)
	require.NotEmpty(t, contents)

	return contents
}

func TestTranslator(t *testing.T) {
	t.Parallel()

	catalog, err := locale.ParseCatalog([]byte("services:\n  live: { fr: \"Actualisé à {time} · toutes les {seconds} s\", en: \"Updated at {time}\" }\n"))
	require.NoError(t, err)

	assert.Equal(t, "Actualisé à 10:41:03 · toutes les 10 s",
		catalog.Translator("fr")("services.live", "time", "10:41:03", "seconds", 10))
	assert.Equal(t, "Updated at {time}", catalog.Translator("en")("services.live"),
		"a parameter that is not given stays as a placeholder")
	assert.Equal(t, "¿services.missing?", catalog.Translator("fr")("services.missing"))
	assert.True(t, catalog.Has("services.live"))
	assert.False(t, catalog.Has("services"), "a section is not an entry")
}

func TestParseCatalogErrors(t *testing.T) {
	t.Parallel()

	_, err := locale.ParseCatalog([]byte("services:\n  title: État des services\n"))
	require.ErrorIs(t, err, locale.ErrCatalogEntry, "a text without languages")

	_, err = locale.ParseCatalog([]byte("services:\n  title: { fr: 10, en: Service status }\n"))
	require.ErrorIs(t, err, locale.ErrCatalogEntry, "a number is not a text")

	_, err = locale.ParseCatalog([]byte("services:\n  title: { fr: État, en: Status, de: Zustand }\n"))
	require.ErrorIs(t, err, locale.ErrCatalogEntry, "an unknown language")

	catalog, err := locale.ParseCatalog([]byte("nav:\n  home: { fr: Accueil, en: Home }\n"))
	require.NoError(t, err)
	require.ErrorIs(t, catalog.Merge(locale.Catalog{"nav.home": {"fr": "x", "en": "y"}}), locale.ErrCatalogDuplicate)
}
