package pesit

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
)

func init() {
	gwtesting.Register(Pesit, gwtesting.ProtoFeatures{
		Protocol: Module{},
		TransID:  true,
		RuleName: false,
		Size:     true,
	})

	gwtesting.Register(PesitTLS, gwtesting.ProtoFeatures{
		Protocol: ModuleTLS{},
		TransID:  true,
		RuleName: false,
		Size:     true,
	})
}

func requireNoError(tb testing.TB, err *pipeline.Error, msgAndArgs ...any) {
	tb.Helper()

	if err != nil {
		assert.FailNow(tb, fmt.Sprintf("Received unexpected error:\n%+v", err), msgAndArgs...)
	}
}

func TestTsInTZ(t *testing.T) {
	t.Parallel()

	loc, err := time.LoadLocation("Europe/Paris")
	require.NoError(t, err)

	t.Run("DST", func(t *testing.T) {
		t.Parallel()

		ts := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
		expected := time.Date(2026, 7, 10, 12, 0, 0, 0, loc).Format(time.RFC3339)
		tsInLoc := tsInTz(ts, loc)

		assert.Equal(t, expected, tsInLoc)
	})

	t.Run("No DST", func(t *testing.T) {
		t.Parallel()

		ts := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
		expected := time.Date(2026, 11, 10, 12, 0, 0, 0, loc).Format(time.RFC3339)
		tsInLoc := tsInTz(ts, loc)

		assert.Equal(t, expected, tsInLoc)
	})
}
