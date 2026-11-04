package backend

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Every list with a filters panel renders the label of its "Filters" button
// inside a live part, so that a filter changed in place refreshes its count
// with the list (#838): the label is never plain text in a template.
func TestFiltersLabelsAreLive(t *testing.T) {
	t.Parallel()

	label := regexp.MustCompile(`gui\.T\(ctx, "filters\.title"\)`)
	live := regexp.MustCompile(`<span id="[a-z-]+-filters" data-live-part="[a-z]+">\{ gui\.T\(ctx, "filters\.title"\)`)

	templates, err := filepath.Glob(filepath.Join("handlers", "*", "*.templ"))
	require.NoError(t, err)
	require.NotEmpty(t, templates)

	lists := 0

	for _, path := range templates {
		content, readErr := os.ReadFile(path)
		require.NoError(t, readErr)

		for i, line := range strings.Split(string(content), "\n") {
			if !label.MatchString(line) {
				continue
			}

			lists++

			assert.Truef(t, live.MatchString(line), "%s:%d: the label of the filters is rendered outside a live part", path, i+1)
		}
	}

	assert.GreaterOrEqual(t, lists, 10, "the lists with a filters panel")
}
