package filewatcher

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"testing/fstest"

	webdavlib "golang.org/x/net/webdav"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/webdav"
)

func init() {
	model.Protocols[webdav.Webdav] = webdav.Module{}
}

func TestListWebDAV(t *testing.T) {
	t.Parallel()

	models := makeTestWebdavModels(t)
	ctx := makeTestContext(t, models)
	lister := newWebdavLister(ctx.logger, ctx.models.asTransCtx(), nil)

	doListerTests(t, lister, 0o664, 0o775)
}

func makeTestWebdavModels(tb testing.TB) *testModels {
	tb.Helper()

	const (
		expectedUsername = "toto"
		expectedPassword = "sesame"
	)

	wdHandler := &webdavlib.Handler{
		FileSystem: testWebdavFS{memFs},
		LockSystem: webdavlib.NewMemLS(),
	}
	handlerFunc := func(w http.ResponseWriter, r *http.Request) {
		user, pswd, ok := r.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic`)
			http.Error(w, "missing credentials", http.StatusUnauthorized)
			return
		}
		if user != expectedUsername {
			http.Error(w,
				fmt.Sprintf("invalid username %q (expected %q)", user, expectedUsername),
				http.StatusUnauthorized)
			return
		}
		if pswd != expectedPassword {
			http.Error(w,
				fmt.Sprintf("invalid password %q (expected %q)", pswd, expectedPassword),
				http.StatusUnauthorized)
			return
		}

		wdHandler.ServeHTTP(w, r)
	}

	server := httptest.NewServer(http.HandlerFunc(handlerFunc))
	tb.Cleanup(func() { server.Close() })

	return &testModels{
		partner: &model.RemoteAgent{
			Name:     "webdav_partner",
			Protocol: webdav.Webdav,
			Address:  types.MustAddr(server.Listener.Addr().String()),
		},
		partnerCredentials: nil,
		user:               &model.RemoteAccount{Login: expectedUsername},
		userCredentials: []*model.Credential{{
			Name:  "password",
			Type:  auth.Password,
			Value: expectedPassword,
		}},
	}
}

type testWebdavFS struct{ fstest.MapFS }

func (testWebdavFS) Mkdir(context.Context, string, os.FileMode) error { return errNotImplemented }
func (testWebdavFS) RemoveAll(context.Context, string) error          { return errNotImplemented }
func (testWebdavFS) Rename(context.Context, string, string) error     { return errNotImplemented }

func (t testWebdavFS) OpenFile(_ context.Context, name string, _ int, _ os.FileMode) (webdavlib.File, error) {
	file, err := t.MapFS.Open(validPath(name))
	if err != nil {
		return nil, err
	}

	return testWebdavFile{file}, nil
}

func (t testWebdavFS) Stat(_ context.Context, name string) (os.FileInfo, error) {
	return t.MapFS.Stat(validPath(name))
}

type testWebdavFile struct{ fs.File }

func (testWebdavFile) Seek(int64, int) (int64, error) { return 0, errNotImplemented }
func (testWebdavFile) Write([]byte) (int, error)      { return 0, errNotImplemented }

func (t testWebdavFile) Readdir(count int) ([]fs.FileInfo, error) {
	return readDirInfoFile(t.File, count)
}
