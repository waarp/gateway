package httptransport

import (
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/analytics"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/logtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
)

func init() {
	analytics.GlobalService = &analytics.Service{}
	SetTestIdleConnTimeout()
}

func TestHTTPTransport(t *testing.T) {
	// Server setup
	var handler testHandler
	server := httptest.NewServer(&handler)
	defer server.Close()

	// Client setup
	transporter, err := NewTransporter(false, "", nil)
	require.NoError(t, err)

	// Test
	testTransporter(t, server, &handler, transporter)
}

func TestHTTPSTransport(t *testing.T) {
	// Server setup
	var handler testHandler
	server := httptest.NewUnstartedServer(&handler)
	server.TLS = &tls.Config{Certificates: []tls.Certificate{gwtesting.ServerCert}}
	server.StartTLS()
	defer server.Close()

	// Client setup
	transporter, err := NewTransporter(true, "", nil)
	require.NoError(t, err)

	// Test
	testTransporter(t, server, &handler, transporter)
}

func testTransporter(tb testing.TB, server *httptest.Server, serverHandler *testHandler,
	transporter *Transporter,
) {
	// Pre-check
	assert.EqualValues(tb, 0, analytics.GlobalService.OpenOutgoingConnections.Load())

	// Setup
	pip := fakePipeline(tb, server)
	rt := transporter.Connect(pip)
	client := &http.Client{Transport: rt}

	// 1st request
	_, err := client.Post(server.URL, "text/plain", nil)
	require.NoError(tb, err)
	assert.EqualValues(tb, 1, analytics.GlobalService.OpenOutgoingConnections.Load())

	// 2nd request
	_, err = client.Post(server.URL, "application/json", nil)
	require.NoError(tb, err)
	assert.EqualValues(tb, 1, analytics.GlobalService.OpenOutgoingConnections.Load())

	// Check requests arrived
	assert.Len(tb, serverHandler.reqs, 2)
	assert.Equal(tb, "text/plain", serverHandler.reqs[0].Header.Get("Content-Type"))
	assert.Equal(tb, "application/json", serverHandler.reqs[1].Header.Get("Content-Type"))

	// Check connection is closed
	<-time.After(idleConnTimeout * 3)
	assert.EqualValues(tb, 0, analytics.GlobalService.OpenOutgoingConnections.Load())
}

func fakePipeline(tb testing.TB, server *httptest.Server) *pipeline.Pipeline {
	tb.Helper()

	u, err := url.Parse(server.URL)
	require.NoError(tb, err)

	pip := &pipeline.Pipeline{
		DB:     dbtest.TestDatabase(tb),
		Logger: logtest.GetTestLogger(tb),
		TransCtx: &model.TransferContext{
			Client: &model.Client{
				Identifier: model.ID(1),
				Name:       "test_client",
				Protocol:   u.Scheme,
			},
			RemoteAgent: &model.RemoteAgent{
				Identifier: model.ID(10),
				Name:       "test_partner",
				Protocol:   u.Scheme,
				Address:    types.MustAddr(u.Host),
			},
			RemoteAccount: &model.RemoteAccount{
				Identifier: model.ID(100),
				Login:      "test_login",
			},
		},
	}
	if u.Scheme == "https" {
		pip.TransCtx.RemoteAgentCreds = []*model.Credential{{
			Type: auth.TLSTrustedCertificate, Value: gwtesting.ServerCertPEM,
		}}
	}

	return pip
}

type testHandler struct {
	reqs []*http.Request
}

func (h *testHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.reqs = append(h.reqs, r)
	io.Copy(io.Discard, r.Body)
	w.WriteHeader(201)
}
