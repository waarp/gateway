package as2test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"os"
	"testing"

	"code.waarp.fr/lib/as2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs/fstest"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/logtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/as2/internal/common"
)

type TestServerCtx struct {
	Root   string
	Addr   string
	Server *as2.Server
	FS     *os.Root
}

func MakeServer(tb testing.TB, isTLS bool, clientCert *x509.Certificate, serverCert *tls.Certificate,
) *TestServerCtx {
	tb.Helper()
	as2ID, clientID := "as2-test-partner", "as2-test-client"
	if isTLS {
		as2ID, clientID = "as2-tls-test-partner", "as2-tls-test-client"
	}

	certOpt := func(optFunc func(*x509.Certificate, any) as2.ServerOption) as2.ServerOption {
		if serverCert != nil {
			return optFunc(serverCert.Leaf, serverCert.PrivateKey)
		}

		return func(*as2.Server) {}
	}
	dir := fstest.TestRoot(tb)

	handleFile := func(_ context.Context, filename string, payload []byte) error {
		return dir.WriteFile(filename, payload, 0o600)
	}

	partnerStore := as2.NewInMemoryPartnerStore()
	require.NoError(tb, partnerStore.Put(tb.Context(), &as2.Partner{
		Name:             clientID,
		AS2ID:            clientID,
		CertChain:        []*x509.Certificate{clientCert},
		RequireSignedMDN: clientCert != nil,
		MDNMode:          as2.MDNModeSync,
		Validator: common.CertValidator{
			Usages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		},
	}))

	logger := logtest.GetTestLogger(tb,
		logtest.WithName("as2-test-server"),
		logtest.WithLevel("WARNING"),
	)
	server := as2.NewServer(
		as2.WithInboundPath("/"),
		as2.WithLogger(logger.Slogger()),
		as2.WithMaxBodyBytes(MaxBodySize),
		as2.WithFileHandler(handleFile),
		as2.WithPartnerStore(partnerStore),
		certOpt(as2.WithMDNSigner),
		certOpt(as2.WithDecryptCertKey),
		as2.WithLocalAS2ID(as2ID),
	)

	list, err := net.Listen("tcp", "localhost:0")
	require.NoError(tb, err)

	if isTLS {
		list = tls.NewListener(list, &tls.Config{
			Certificates: []tls.Certificate{*serverCert},
		})
	}

	//nolint:errcheck //we don't care about the error here
	go server.Serve(list)

	tb.Cleanup(func() {
		assert.NoError(tb, server.Shutdown(context.Background()))
	})

	return &TestServerCtx{
		Root:   dir.Name(),
		Addr:   list.Addr().String(),
		Server: server,
		FS:     dir,
	}
}

func (s *TestServerCtx) ReadFile(tb testing.TB, filename string) []byte {
	tb.Helper()
	data, err := s.FS.ReadFile(filename)
	require.NoError(tb, err)

	return data
}
