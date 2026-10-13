package azure

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseOpts(t *testing.T) {
	t.Parallel()

	opts := map[string]string{}
	parseAzureOpts("account", "password", opts)

	assert.NotContains(t, opts, "account")
	assert.NotContains(t, opts, "key")
}
