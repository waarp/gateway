package filewatcher

import (
	"fmt"
	"net"
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66/r66auth"
	r66lib "code.waarp.fr/lib/r66"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func init() {
	model.Protocols[r66.R66] = r66.Module{}
}

func TestListR66(t *testing.T) {
	t.Parallel()

	models := makeTestR66Models(t)
	ctx := makeTestContext(t, models)
	lister := newR66Lister(ctx.logger, ctx.models.asTransCtx(), ctx.dialer, nil)

	doListerTests(t, lister, 0, 0)
}

func makeTestR66Models(tb testing.TB) *testModels {
	tb.Helper()

	const (
		expectedUsername = "toto"
		expectedPassword = "sesame"

		serverLogin    = "r66_server"
		serverPassword = "p@ssw0rd"
	)

	server := r66lib.Server{
		Login:    serverLogin,
		Password: []byte(serverPassword),
		Handler: testR66Handler{
			expectedLogin:    expectedUsername,
			expectedPassword: r66auth.CryptPass(expectedPassword),
		},
		Conf: &r66lib.Config{},
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(tb, err)
	tb.Cleanup(func() { listener.Close() })

	go server.Serve(listener)
	tb.Cleanup(func() { server.Shutdown(tb.Context()) })

	return &testModels{
		partner: &model.RemoteAgent{
			Name:     serverLogin,
			Protocol: r66.R66,
			Address:  types.MustAddr(listener.Addr().String()),
		},
		partnerCredentials: []*model.Credential{{
			Type:  auth.Password,
			Value: serverPassword,
		}},
		user: &model.RemoteAccount{Login: expectedUsername},
		userCredentials: []*model.Credential{{
			Type:  auth.Password,
			Value: expectedPassword,
		}},
	}
}

type testR66Handler struct {
	expectedLogin    string
	expectedPassword string
}

func (t testR66Handler) ValidRequest(*r66lib.Request) (r66lib.TransferHandler, error) {
	return nil, errNotImplemented
}

func (t testR66Handler) ValidAuth(auth *r66lib.Authent) (r66lib.SessionHandler, error) {
	if auth.Login != t.expectedLogin {
		return nil, fmt.Errorf("invalid login %q (expected %q)", auth.Login, t.expectedLogin)
	}
	if pswd := string(auth.Password); pswd != t.expectedPassword {
		return nil, fmt.Errorf("invalid password %q (expected %q)", pswd, t.expectedPassword)
	}

	return t, nil
}

func (testR66Handler) GetTransferInfo(int64, bool) (*r66lib.TransferInfo, error) {
	return nil, errNotImplemented
}

func (testR66Handler) GetFileInfo(_ string, pattern string) ([]r66lib.FileInfo, error) {
	matches, globErr := memFs.Glob(pattern)
	if globErr != nil {
		return nil, globErr
	}

	r66Info := make([]r66lib.FileInfo, len(matches))
	for i, name := range matches {
		f, _ := memFs.Stat(name)

		r66Info[i] = r66lib.FileInfo{
			Name:       f.Name(),
			Size:       f.Size(),
			LastModify: f.ModTime(),
			Type:       utils.If(f.IsDir(), "directory", "file"),
			Permission: f.Mode().Perm().String(),
		}
	}

	return r66Info, globErr
}
