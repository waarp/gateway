package rules

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/locale"
)

// The types to choose from show their name first and come sorted by name
// within their family (2026-10-29): the operators know the tasks by their
// names, from the command line and the documentation.
func TestTypeChoices(t *testing.T) {
	t.Parallel()

	catalog, err := locale.LoadCatalog()
	require.NoError(t, err)

	translate := catalog.Translator("fr")
	groups := typeChoices(translate, "EXEC")
	require.Len(t, groups, len(families))

	var byFamily [][]string

	for _, group := range groups {
		codes := make([]string, 0, len(group.Choices))

		for _, choice := range group.Choices {
			codes = append(codes, choice.Code)
			assert.Equal(t, choice.Value, choice.Code, "the code of %s is its name", choice.Value)
			assert.Equal(t, typeLabel(translate, choice.Value), choice.Label, "the label of %s", choice.Value)
			assert.NotEmpty(t, choice.Text, "what %s does", choice.Value)
			assert.Equal(t, choice.Value == "EXEC", choice.Selected, "the chosen one, %s", choice.Value)
		}

		assert.True(t, slices.IsSorted(codes), "%s sorted by name: %s", group.Label, strings.Join(codes, ", "))
		byFamily = append(byFamily, codes)
	}

	assert.Len(t, taskSpecs, len(slices.Concat(byFamily...)), "every type offered once")
	assert.Equal(t, "Opérations de fichiers", groups[0].Label)
	assert.Equal(t, []string{"CHMOD", "COPY", "COPYRENAME", "DELETE", "MOVE", "MOVERENAME", "REMOTEDELETE"}, byFamily[0],
		"the file operations, by name")
}
