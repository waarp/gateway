package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The rail of the collapsed sidebar shows the icons alone: every screen has
// one of its own (#772).
func TestNavigationIconsAreDistinct(t *testing.T) {
	t.Parallel()

	screens := map[string]string{}

	for _, group := range navigation() {
		for _, entry := range group.entries {
			assert.NotContains(t, screens, entry.icon, "the icon of %s is also the icon of %s", entry.key, screens[entry.icon])
			screens[entry.icon] = entry.key
		}
	}
}
