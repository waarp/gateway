package pipeline

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// Once a transfer has ended, the pipeline's error guard is consumed without
// an error (EndTransfer). A file operation failing afterwards, from a data
// goroutine still running, must still get an error: a nil *Error returned in
// an error interface is not nil, and the caller would dereference it.
func TestFileStreamErrorAfterEnd(t *testing.T) {
	t.Parallel()

	pip := &Pipeline{}
	pip.errOnce.Do(func() {}) // consumed without an error, as EndTransfer does

	err := pip.internalError(types.TeInternal, "failed to write file", nil)
	require.NotNil(t, err, "a stopped pipeline still hands out an error")
	assert.Equal(t, types.TeInternal, err.Code())
	assert.Contains(t, err.Error(), "failed to write file")

	stream := &FileStream{Pipeline: pip}

	sErr := stream.internalErrorWithMsg(types.TeInternal, "write trace error",
		"failed to write file", nil)
	require.NotNil(t, sErr, "a stopped file stream still hands out an error")
	assert.Equal(t, types.TeInternal, sErr.Code())

	// What a caller such as io.Copy gets back.
	var asErr error = sErr

	require.NotNil(t, asErr)
	assert.NotPanics(t, func() { _ = asErr.Error() })
}
