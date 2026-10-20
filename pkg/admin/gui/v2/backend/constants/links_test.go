package constants

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDocPage(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "https://doc.waarp.org/waarp-gateway/latest/fr/user/webui/etat-des-services/",
		DocPage("fr", "user/webui/etat-des-services"))
	assert.Equal(t, "https://doc.waarp.org/waarp-gateway/latest/fr/user/webui/etat-des-services/",
		DocPage("fr", "/user/webui/etat-des-services/"), "the scheme and the slashes are kept as they are")
}
