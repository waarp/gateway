package filewatcher

import (
	"crypto/tls"
	"fmt"
	"io/fs"
	"net"
	"os"
	"testing"
	"time"

	ftplib "github.com/fclairamb/ftpserverlib"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ftp"
)

func init() {
	model.Protocols[ftp.FTP] = ftp.Module{}
}

func TestListFTP(t *testing.T) {
	t.Parallel()

	models := makeTestFTPModels(t)
	ctx := makeTestContext(t, models)
	lister := newFTPLister(ctx.logger, ctx.models.asTransCtx(), nil)

	doListerTests(t, lister, 0, 0)
}

func makeTestFTPModels(tb testing.TB) *testModels {
	tb.Helper()

	const (
		expectedUsername = "toto"
		expectedPassword = "sesame"
	)

	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(tb, err)

	handler := &testFTPHandler{
		listener:         listener,
		expectedLogin:    expectedUsername,
		expectedPassword: expectedPassword,
	}

	server := ftplib.NewFtpServer(handler)
	require.NoError(tb, server.Listen())
	go server.Serve()
	tb.Cleanup(func() { server.Stop() })

	return &testModels{
		partner: &model.RemoteAgent{
			Name:     "ftp_server",
			Protocol: ftp.FTP,
			Address:  types.MustAddr(server.Addr()),
		},
		partnerCredentials: nil,
		user:               &model.RemoteAccount{Login: expectedUsername},
		userCredentials: []*model.Credential{{
			Type:  auth.Password,
			Value: expectedPassword,
		}},
	}
}

type testFTPHandler struct {
	listener         net.Listener
	expectedLogin    string
	expectedPassword string
}

func (t testFTPHandler) GetSettings() (*ftplib.Settings, error) {
	return &ftplib.Settings{
		Listener: t.listener,
		// FTP server lib does not return permissions when using MLSD. So we
		// disable that command to use LIST instead which does return permissions.
		DisableMLSD: true,
	}, nil
}

func (testFTPHandler) ClientConnected(ftplib.ClientContext) (string, error) {
	return "Test FTP Server", nil
}
func (testFTPHandler) ClientDisconnected(ftplib.ClientContext) {}
func (testFTPHandler) GetTLSConfig() (*tls.Config, error)      { return nil, errNotImplemented }

func (t testFTPHandler) AuthUser(_ ftplib.ClientContext, user, pass string) (ftplib.ClientDriver, error) {
	if user != t.expectedLogin {
		return nil, fmt.Errorf("invalid login %q (expected %q)", user, t.expectedLogin)
	}
	if pass != t.expectedPassword {
		return nil, fmt.Errorf("invalid password %q (expected %q)", pass, t.expectedPassword)
	}
	return t, nil
}

func (testFTPHandler) Name() string                               { return "Test FTP Server" }
func (testFTPHandler) Create(string) (afero.File, error)          { return nil, errNotImplemented }
func (testFTPHandler) Mkdir(string, os.FileMode) error            { return errNotImplemented }
func (testFTPHandler) MkdirAll(string, os.FileMode) error         { return errNotImplemented }
func (testFTPHandler) Remove(string) error                        { return errNotImplemented }
func (testFTPHandler) Chmod(string, os.FileMode) error            { return errNotImplemented }
func (testFTPHandler) Chown(string, int, int) error               { return errNotImplemented }
func (testFTPHandler) Chtimes(string, time.Time, time.Time) error { return errNotImplemented }
func (testFTPHandler) RemoveAll(string) error                     { return errNotImplemented }
func (testFTPHandler) Rename(string, string) error                { return errNotImplemented }
func (testFTPHandler) OpenFile(string, int, os.FileMode) (afero.File, error) {
	return nil, errNotImplemented
}

func (testFTPHandler) Open(name string) (afero.File, error) {
	name = validPath(name)
	file, err := memFs.Open(name)
	if err != nil {
		return nil, err
	}

	return testFTPFile{file, name}, nil
}

func (testFTPHandler) Stat(name string) (os.FileInfo, error) {
	return memFs.Stat(validPath(name))
}

type testFTPFile struct {
	fs.File
	name string
}

func (testFTPFile) Close() error                       { return nil }
func (testFTPFile) Read([]byte) (int, error)           { return 0, errNotImplemented }
func (testFTPFile) ReadAt([]byte, int64) (int, error)  { return 0, errNotImplemented }
func (testFTPFile) Write([]byte) (int, error)          { return 0, errNotImplemented }
func (testFTPFile) WriteAt([]byte, int64) (int, error) { return 0, errNotImplemented }
func (testFTPFile) Seek(int64, int) (int64, error)     { return 0, errNotImplemented }
func (testFTPFile) Sync() error                        { return errNotImplemented }
func (testFTPFile) Truncate(int64) error               { return errNotImplemented }
func (testFTPFile) WriteString(string) (int, error)    { return 0, errNotImplemented }

func (t testFTPFile) Name() string { return t.name }

func (t testFTPFile) readdir(count int) ([]fs.DirEntry, error) {
	readDirFile, ok := t.File.(fs.ReadDirFile)
	if !ok {
		return nil, errNotImplemented
	}

	return readDirFile.ReadDir(count)
}

func (t testFTPFile) Readdir(count int) ([]os.FileInfo, error) {
	entries, err := t.readdir(count)
	if err != nil {
		return nil, err
	}

	infos := make([]os.FileInfo, len(entries))
	for i, entry := range entries {
		if infos[i], err = entry.Info(); err != nil {
			return infos, err
		}
	}

	return infos, nil
}

func (t testFTPFile) Stat() (os.FileInfo, error) {
	return t.File.Stat()
}

func (t testFTPFile) Readdirnames(count int) ([]string, error) {
	entries, err := t.readdir(count)
	if err != nil {
		return nil, err
	}

	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}

	return names, nil
}
