package pesit

import (
	"testing"

	"code.waarp.fr/lib/pesit"
	"github.com/stretchr/testify/assert"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
)

// Once DeselectFile has reset the handler for the next transfer, a late error
// coming from the data exchange must not make the server panic.
func TestHandleErrorAfterReset(t *testing.T) {
	t.Parallel()

	handler := &transferHandler{logger: gwtesting.Logger(t)} // no transfer: pip is nil

	for _, err := range []error{
		pesit.NewDiagnostic(pesit.CodeInternalError, "late error"),
		pesit.NewDiagnostic(pesit.CodeVolontaryTermination, "late cancel"),
		pesit.NewDiagnostic(pesit.CodeTryLater, "late pause"),
	} {
		assert.NotPanics(t, func() { handler.handleError(err) })
	}
}
