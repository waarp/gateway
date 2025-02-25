package tasks

import (
	"io"
	"net"
	"net/http"
	"net/textproto"
	"strconv"
	"testing"

	"github.com/egirna/icap"
	ic "github.com/solidwall/icap-client"
	"github.com/stretchr/testify/require"
)

type testHTTPRequest struct {
	url     string
	method  string
	headers http.Header
	data    []byte
}

type testHTTPResponse struct {
	code    int
	headers http.Header
	data    []byte
}

type testIcapReqModRequest struct {
	icapHeaders textproto.MIMEHeader
	payload     testHTTPRequest
}

type testIcapRespModRequest struct {
	icapHeaders textproto.MIMEHeader
	payload     testHTTPResponse
}

func readBody(tb testing.TB, r io.ReadCloser) string {
	tb.Helper()

	b, err := io.ReadAll(r)
	require.NoError(tb, err)
	require.NoError(tb, r.Close())

	return string(b)
}

func makeReqModServer(tb testing.TB, expReq *testIcapReqModRequest,
	responseCode int,
) string {
	tb.Helper()

	if expReq.icapHeaders == nil {
		expReq.icapHeaders = make(textproto.MIMEHeader)
	}

	if expReq.payload.headers == nil {
		expReq.payload.headers = make(http.Header)
	}

	previewLen := len(expReq.payload.data)

	handler := func(w icap.ResponseWriter, r *icap.Request) {
		switch r.Method {
		case ic.MethodOPTIONS: // OPTIONS
			w.Header().Set(ic.PreviewHeader, strconv.Itoa(previewLen))
			w.Header().Set(ic.TransferPreviewHeader, "*")
			w.WriteHeader(http.StatusOK, nil, false)
		case ic.MethodREQMOD: // TRANSFER
			require.NotNil(tb, r.Request, "missing HTTP request")

			for head, val := range expReq.icapHeaders {
				require.Equal(tb, val, r.Header.Get(head), "wrong ICAP header")
			}

			for head, val := range expReq.payload.headers {
				require.Equal(tb, val, r.Request.Header.Get(head), "wrong HTTP request header")
			}

			require.Equal(tb, expReq.payload.method, r.Request.Method, "wrong HTTP request method")
			require.Equal(tb, expReq.payload.url, r.Request.RequestURI, "wrong HTTP request URL")

			body := readBody(tb, r.Request.Body)
			require.Equal(tb, string(expReq.payload.data), body, "wrong HTTP request body")

			w.WriteHeader(responseCode, nil, false)
		default:
			tb.Fatalf("unexpected ICAP method %q", r.Method)
		}
	}

	list, listErr := net.Listen("tcp", "localhost:0")
	require.NoError(tb, listErr, "failed to start listener")
	tb.Cleanup(func() {
		require.NoError(tb, list.Close(), "failed to close listener")
	})

	go icap.Serve(list, icap.HandlerFunc(handler))

	return list.Addr().String()
}

func makeRespModServer(tb testing.TB, expReq *testIcapRespModRequest,
	responseCode int,
) string {
	tb.Helper()

	if expReq.icapHeaders == nil {
		expReq.icapHeaders = make(textproto.MIMEHeader)
	}

	if expReq.payload.headers == nil {
		expReq.payload.headers = make(http.Header)
	}

	previewLen := len(expReq.payload.data)

	handler := func(w icap.ResponseWriter, r *icap.Request) {
		switch r.Method {
		case ic.MethodOPTIONS: // OPTIONS
			w.Header().Set(ic.PreviewHeader, strconv.Itoa(previewLen))
			w.Header().Set(ic.TransferPreviewHeader, "*")
			w.WriteHeader(http.StatusOK, nil, true)
		case ic.MethodRESPMOD: // TRANSFER
			require.NotNil(tb, r.Response, "missing HTTP request")

			for head, val := range expReq.icapHeaders {
				require.Equal(tb, val, r.Header.Get(head), "wrong ICAP header")
			}

			for head, val := range expReq.payload.headers {
				require.Equal(tb, val, r.Response.Header.Get(head), "wrong HTTP response header")
			}

			require.Equal(tb, expReq.payload.code, r.Response.StatusCode, "wrong HTTP response code")

			body := readBody(tb, r.Response.Body)
			require.Equal(tb, string(expReq.payload.data), body, "wrong HTTP response body")

			w.WriteHeader(responseCode, nil, false)
		default:
			tb.Fatalf("unexpected ICAP method %q", r.Method)
		}
	}

	list, listErr := net.Listen("tcp", "localhost:0")
	require.NoError(tb, listErr, "failed to start listener")
	tb.Cleanup(func() {
		require.NoError(tb, list.Close(), "failed to close listener")
	})

	go icap.Serve(list, icap.HandlerFunc(handler))

	return list.Addr().String()
}
