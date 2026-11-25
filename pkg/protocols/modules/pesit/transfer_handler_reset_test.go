package pesit

import (
	"io"
	"strings"
	"testing"

	"code.waarp.fr/lib/pesit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

type cancelledSender struct{ err error }

func (cancelledSender) SetManualArticleHandling(bool) bool { return true }

func (c cancelledSender) StartNextSendArticle() (io.Writer, error) { return nil, c.err }

// A partner cancel with diagnostic 0 reaches the emission as a success-coded
// error that comes without an article: the emission ends there.
func TestMultiSendAfterSuccessfulCancel(t *testing.T) {
	t.Parallel()

	handler := &transferHandler{logger: gwtesting.Logger(t), pip: &pipeline.Pipeline{}}
	cancelled := cancelledSender{err: pesit.NewDiagnostic(pesit.CodeSuccess, "")}

	var err error

	require.NotPanics(t, func() {
		err = handler.multiSend(cancelled, strings.NewReader("first article"), newArticleLengths(13))
	})
	require.NoError(t, err)
}
