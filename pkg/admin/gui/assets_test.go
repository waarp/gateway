package gui

import (
	"io/fs"
	"path"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/frontend"
)

var (
	cssFontFace = regexp.MustCompile(`(?s)@font-face\s*\{[^}]*\}`)
	cssURL      = regexp.MustCompile(`url\(\s*['"]?([^'")]+)['"]?\s*\)`)
)

// TestStylesheetsReferenceEmbeddedFiles checks that what the embedded
// stylesheets of the web interface reference is embedded too: a stylesheet
// imported, an image, and for each font at least one of its sources (the
// browser takes the first one it supports). A missing font draws an empty
// square instead of each icon.
func TestStylesheetsReferenceEmbeddedFiles(t *testing.T) {
	t.Parallel()

	historic, err := fs.Sub(webFS, "front-end")
	require.NoError(t, err)

	for name, fsys := range map[string]fs.FS{"historic": historic, "v2": frontend.EmbedFS} {
		require.NoError(t, fs.WalkDir(fsys, ".", func(file string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || path.Ext(file) != ".css" {
				return err
			}

			data, err := fs.ReadFile(fsys, file)
			if err != nil {
				return err
			}

			exists := func(ref string) bool {
				_, statErr := fs.Stat(fsys, path.Join(path.Dir(file), ref))

				return statErr == nil
			}

			css := string(data)
			for _, face := range cssFontFace.FindAllString(css, -1) {
				refs := local(cssURL.FindAllStringSubmatch(face, -1))
				assert.Truef(t, len(refs) == 0 || anyOf(refs, exists),
					"%s %s: no source of a font is embedded among %v", name, file, refs)
			}

			for _, ref := range local(cssURL.FindAllStringSubmatch(cssFontFace.ReplaceAllString(css, ""), -1)) {
				assert.Truef(t, exists(ref), "%s %s references %q, which is not embedded", name, file, ref)
			}

			return nil
		}))
	}
}

// local keeps the references to files of the assets: neither inline data nor
// another site, without the query of a cache buster.
func local(matches [][]string) []string {
	var refs []string

	for _, m := range matches {
		ref := m[1]
		if strings.HasPrefix(ref, "data:") || strings.Contains(ref, "://") || strings.HasPrefix(ref, "#") {
			continue
		}

		if i := strings.IndexAny(ref, "?#"); i >= 0 {
			ref = ref[:i]
		}

		refs = append(refs, ref)
	}

	return refs
}

func anyOf(refs []string, ok func(string) bool) bool {
	for _, ref := range refs {
		if ok(ref) {
			return true
		}
	}

	return false
}
