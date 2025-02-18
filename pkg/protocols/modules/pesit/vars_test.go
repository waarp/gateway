package pesit

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
)

func init() {
	gwtesting.Protocols[Pesit] = gwtesting.ProtoFeatures{
		MakeClient:        Module{}.NewClient,
		MakeServer:        Module{}.NewServer,
		MakeServerConfig:  Module{}.MakeServerConfig,
		MakeClientConfig:  Module{}.MakeClientConfig,
		MakePartnerConfig: Module{}.MakePartnerConfig,
		TransID:           true,
		RuleName:          false,
		Size:              true,
	}

	gwtesting.Protocols[PesitTLS] = gwtesting.ProtoFeatures{
		MakeClient:        ModuleTLS{}.NewClient,
		MakeServer:        ModuleTLS{}.NewServer,
		MakeServerConfig:  ModuleTLS{}.MakeServerConfig,
		MakeClientConfig:  ModuleTLS{}.MakeClientConfig,
		MakePartnerConfig: ModuleTLS{}.MakePartnerConfig,
		TransID:           true,
		RuleName:          false,
		Size:              true,
	}
}

func requireNoError(tb testing.TB, err *pipeline.Error, msgAndArgs ...interface{}) {
	tb.Helper()

	if err != nil {
		assert.FailNow(tb, fmt.Sprintf("Received unexpected error:\n%+v", err), msgAndArgs...)
	}
}
