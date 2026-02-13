package webdav_test

import (
	"crypto/rand"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"os"
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"github.com/stretchr/testify/require"
	"github.com/studio-b12/gowebdav"
	weblib "golang.org/x/net/webdav"

	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/webdav"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
)

const (
	buffSize = 2000000 // 2MB
	password = "sesame"
)

func makeTransport(ctx *gwtesting.TestServerCtx) http.RoundTripper {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DisableKeepAlives = true

	if ctx.Server.Protocol == webdav.WebdavTLS {
		rootCAs := utils.TLSCertPool()
		rootCAs.AddCert(gwtesting.LocalhostCert.Leaf)
		transport.TLSClientConfig = &tls.Config{RootCAs: rootCAs}
	}

	return transport
}

func makeClient(ctx *gwtesting.TestServerCtx) *gowebdav.Client {
	scheme := "http://"
	if ctx.Server.Protocol == webdav.WebdavTLS {
		scheme = "https://"
	}

	client := gowebdav.NewClient(scheme+ctx.Server.Address.String(),
		ctx.Account.Login, password)
	client.SetTransport(makeTransport(ctx))

	return client
}

func makeBuf(tb testing.TB) []byte {
	buf := make([]byte, buffSize)
	_, err := io.ReadFull(rand.Reader, buf)
	require.NoError(tb, err)

	return buf
}

type servCtx struct {
	addr string
	fs   weblib.FileSystem
}

func makeServer(tb testing.TB) *servCtx {
	fs := weblib.Dir(tb.TempDir())

	list, err := net.Listen("tcp", "localhost:0")
	require.NoError(tb, err)

	handler := &weblib.Handler{
		Prefix:     "",
		FileSystem: fs,
		LockSystem: weblib.NewMemLS(),
		Logger: func(r *http.Request, err error) {
			if err != nil {
				tb.Errorf("Webdav %s request to %q failed: %v", r.Method, r.URL, err)
			} else {
				tb.Logf("Webdav %s request to %q processed", r.Method, r.URL)
			}
		},
	}

	const expLogin = "test-partner-account"
	serv := http.Server{
		Addr: list.Addr().String(),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			login, pswd, ok := r.BasicAuth()
			if !ok || login != expLogin || pswd != password {
				w.Header().Add("WWW-Authenticate", "Basic")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			handler.ServeHTTP(w, r)
		}),
	}

	go serv.Serve(list)
	tb.Cleanup(func() {
		require.NoError(tb, serv.Shutdown(tb.Context()))
		require.NoError(tb, serv.Close())
	})

	return &servCtx{addr: serv.Addr, fs: fs}
}

func (s *servCtx) readFile(tb testing.TB, path string) []byte {
	file, err := s.fs.OpenFile(tb.Context(), path, os.O_RDONLY, 0)
	require.NoError(tb, err)
	defer file.Close()

	cont, err := io.ReadAll(file)
	require.NoError(tb, err)

	return cont
}

func (s *servCtx) writeFile(tb testing.TB, path string, content []byte) {
	file, err := s.fs.OpenFile(tb.Context(), path, os.O_WRONLY|os.O_CREATE, 0o644)
	require.NoError(tb, err)
	defer file.Close()

	_, err = file.Write(content)
	require.NoError(tb, err)
}
