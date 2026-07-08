package as2_test

import (
	"crypto/rand"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/as2"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/http/httptransport"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
)

func TestMain(m *testing.M) {
	httptransport.SetTestIdleConnTimeout()
	gwtesting.Register(as2.AS2, gwtesting.ProtoFeatures{
		Protocol: as2.Module{},
		TransID:  true,
		RuleName: true,
		Size:     true,
	})
	gwtesting.Register(as2.AS2TLS, gwtesting.ProtoFeatures{
		Protocol: as2.ModuleTLS{},
		TransID:  true,
		RuleName: true,
		Size:     true,
	})

	m.Run()
}

const buffSize = 1_000_000

func makeBuf(tb testing.TB) []byte {
	buf := make([]byte, buffSize)
	_, err := io.ReadFull(rand.Reader, buf)
	require.NoError(tb, err)

	return buf
}
