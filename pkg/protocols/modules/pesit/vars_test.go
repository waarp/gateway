package pesit

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

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

func requireNoError(tb testing.TB, err *pipeline.Error, msgAndArgs ...interface{}) {
	tb.Helper()

	if err != nil {
		assert.FailNow(tb, fmt.Sprintf("Received unexpected error:\n%+v", err), msgAndArgs...)
	}
}
