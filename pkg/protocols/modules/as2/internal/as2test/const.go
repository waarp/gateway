package as2test

import (
	"crypto/rand"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/as2"
)

const (
	TestFileSize = 1000 * 1000      // 1MB
	MaxBodySize  = TestFileSize * 2 // 2MB for safety

	EncryptAlgo = as2.EncryptAlgoAES256CBC

	Password = "sesame"
)

func makeBuf(tb testing.TB, size int) []byte {
	tb.Helper()

	buf := make([]byte, size)
	_, err := io.ReadFull(rand.Reader, buf)
	require.NoError(tb, err)

	return buf
}
