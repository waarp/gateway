package pesit

import (
	"io"
	"testing"

	"code.waarp.fr/lib/pesit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
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

type endedArticles struct{ err error }

func (e endedArticles) GetNextRecvArticle() (io.Reader, error) { return nil, e.err }

// When the data exchange only runs after the connection was released, the
// library hands it the success diagnostic of the release instead of an
// article: the reception is over, there is nothing left to read.
func TestReceiveAfterSuccessfulRelease(t *testing.T) {
	t.Parallel()

	handler := &transferHandler{
		logger: gwtesting.Logger(t),
		pip: &pipeline.Pipeline{TransCtx: &model.TransferContext{
			Transfer: &model.Transfer{TransferInfo: map[string]any{}},
		}},
	}
	released := endedArticles{err: pesit.NewDiagnostic(pesit.CodeSuccess, "")}

	var err error

	require.NotPanics(t, func() { err = handler.receiveTransfer(released) })
	require.NoError(t, err)
}
