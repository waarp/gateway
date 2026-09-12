package pesit

import (
	"testing"

	"code.waarp.fr/lib/pesit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

func TestToPesitErrWithANilTypedError(t *testing.T) {
	t.Parallel()

	// A nil *pipeline.Error stored in an error interface is not nil: a caller
	// checking err != nil hands it over, and the conversion must not panic.
	var pErr *pipeline.Error

	var err error = pErr

	require.False(t, err == nil, "a nil pointer in an error interface is not a nil error")

	var diag pesit.Diagnostic

	require.NotPanics(t, func() { diag = toPesitErr(pesit.CodeInternalError, err) })
	assert.Equal(t, pesit.CodeInternalError, diag.GetCode())
}
