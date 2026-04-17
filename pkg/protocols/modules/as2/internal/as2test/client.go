package as2test

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"slices"
	"testing"

	"code.waarp.fr/lib/as2"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/logging/logtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
)

type TestClientCtx struct {
	FileContent []byte
	Client      *as2.Client
	Partner     *as2.Partner

	isTLS      bool
	clientCert *tls.Certificate
	serverCert *x509.Certificate
}

func MakeClient(tb testing.TB, isTLS bool, serverAddr string,
	clientCert *tls.Certificate, serverCert *x509.Certificate,
) *TestClientCtx {
	tb.Helper()

	serverPartnerName, accountID := "as2-test-server", "as2-test-account"
	if isTLS {
		serverPartnerName, accountID = "as2-tls-test-server", "as2-tls-test-account"
	}

	logger := gwtesting.Logger(tb,
		logtest.WithName("as2-test-client"),
		logtest.WithLevel("WARNING"),
	)
	client := as2.NewClient(
		as2.WithClientLogger(logger.Slogger()),
		as2.WithMDNAddress("foo@bar.org"),
		as2.WithLocalClientAS2ID(accountID),
		func(client *as2.Client) {
			if clientCert != nil {
				as2.WithSigner(clientCert.PrivateKey, parseCertChain(tb, clientCert))(client)
			}

			if isTLS {
				rootCAs := x509.NewCertPool()
				rootCAs.AddCert(serverCert)
				tlsConfig := &tls.Config{RootCAs: rootCAs}
				transport := &http.Transport{TLSClientConfig: tlsConfig}
				httpClient := &http.Client{Transport: transport}
				as2.WithHTTPClient(httpClient)(client)
			}
		},
	)

	serverURL := "http://" + serverAddr
	if isTLS {
		serverURL = "https://" + serverAddr
	}

	partner := as2.Partner{
		Name:    serverPartnerName,
		AS2ID:   serverPartnerName,
		SendUrl: serverURL,
	}
	if serverCert != nil {
		partner.CertChain = []*x509.Certificate{serverCert}
	}

	return &TestClientCtx{
		FileContent: makeBuf(tb, TestFileSize),
		Client:      client,
		Partner:     &partner,
		isTLS:       isTLS,
		clientCert:  clientCert,
		serverCert:  serverCert,
	}
}

func (ctx *TestClientCtx) Run(tb testing.TB, filename string) *as2.SendResult {
	tb.Helper()

	sendOpts := as2.NewSendOptions(
		as2.WithFileName(filename),
		func(opts *as2.SendOptions) {
			if ctx.serverCert != nil {
				as2.WithEncryptFunc(as2.NewPKCS7Encrypter(EncryptAlgo.PKCS7()))(opts)
			} else {
				as2.WithSignedMDN(false)(opts)
			}
		},
	)

	res, err := ctx.Client.Send(tb.Context(), ctx.Partner, ctx.FileContent, sendOpts)
	require.NoError(tb, err)

	return res
}

func parseCertChain(tb testing.TB, cert *tls.Certificate) []*x509.Certificate {
	tb.Helper()

	der := slices.Concat(cert.Certificate...)
	certs, err := x509.ParseCertificates(der)
	require.NoError(tb, err)

	return certs
}
