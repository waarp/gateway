package format

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/locale"
)

// translators are the French and English translators of the real catalogue:
// the shapes below are the ones the screens show.
func translators(t *testing.T) (fr, en func(string, ...any) string) {
	t.Helper()

	catalog, err := locale.LoadCatalog()
	require.NoError(t, err)

	return catalog.Translator("fr"), catalog.Translator("en")
}

func TestGroupDigits(t *testing.T) {
	t.Parallel()

	fr, en := translators(t)

	assert.Equal(t, "0", GroupDigits(fr, 0))
	assert.Equal(t, "999", GroupDigits(fr, 999))
	assert.Equal(t, "1 000", GroupDigits(fr, 1000), "a narrow no-break space in French")
	assert.Equal(t, "10 485 760", GroupDigits(fr, 10485760))
	assert.Equal(t, "10,485,760", GroupDigits(en, 10485760), "a comma in English")
}

func TestBytes(t *testing.T) {
	t.Parallel()

	fr, en := translators(t)

	assert.Equal(t, "0 octet", Bytes(fr, 0))
	assert.Equal(t, "1 octet", Bytes(fr, 1))
	assert.Equal(t, "512 octets", Bytes(fr, 512))
	assert.Equal(t, "1,0 Ko", Bytes(fr, 1024))
	assert.Equal(t, "1,2 Mo", Bytes(fr, 1228800))
	assert.Equal(t, "10 Mo", Bytes(fr, 10485760))
	assert.Equal(t, "2,0 To", Bytes(fr, 2<<40))
	assert.Equal(t, "1.2 MB", Bytes(en, 1228800))
}

func TestExactBytes(t *testing.T) {
	t.Parallel()

	fr, _ := translators(t)

	assert.Equal(t, "1 octet", ExactBytes(fr, 1))
	assert.Equal(t, "1 228 800 octets", ExactBytes(fr, 1228800))
}

func TestWholeBytes(t *testing.T) {
	t.Parallel()

	fr, en := translators(t)

	// Under a kibibyte, or not a whole number of a unit: the exact size alone.
	assert.Equal(t, "512 octets", WholeBytes(fr, 512))
	assert.Equal(t, "32 700 octets", WholeBytes(fr, 32700))
	assert.Equal(t, "65 535 octets", WholeBytes(fr, 65535))
	// A whole number of a unit: the unit first, the exact size after.
	assert.Equal(t, "64 Ko (65 536 octets)", WholeBytes(fr, 65536))
	assert.Equal(t, "1 Mo (1 048 576 octets)", WholeBytes(fr, 1048576))
	assert.Equal(t, "10 Mo (10 485 760 octets)", WholeBytes(fr, 10485760))
	assert.Equal(t, "1 536 Ko (1 572 864 octets)", WholeBytes(fr, 1572864), "1.5 MiB is a whole number of KiB only")
	assert.Equal(t, "10 MB (10,485,760 bytes)", WholeBytes(en, 10485760))
}
