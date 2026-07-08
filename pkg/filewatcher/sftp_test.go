package filewatcher

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"testing"

	sftplib "github.com/pkg/sftp"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func init() {
	model.Protocols[sftp.SFTP] = sftp.Module{}
}

func TestListSFTP(t *testing.T) {
	t.Parallel()

	models := makeTestSFTPModels(t)
	ctx := makeTestContext(t, models)
	lister := newSFTPLister(ctx.logger, ctx.models.asTransCtx(), ctx.dialer, nil)

	doListerTests(t, lister, 0, 0)
}

func makeTestSFTPModels(tb testing.TB) *testModels {
	tb.Helper()

	const (
		hostkeyPK        = testhelpers.RSAPk
		hostkeyPBK       = testhelpers.SSHPbk
		expectedUsername = "toto"
		expectedPassword = "sesame"
	)

	hostkey, err := ssh.ParsePrivateKey([]byte(hostkeyPK))
	require.NoError(tb, err)

	serverConfig := &ssh.ServerConfig{
		PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if user := conn.User(); user != expectedUsername {
				return nil, fmt.Errorf("incorrect username %q (expected %q)", user, expectedUsername)
			}
			if pswd := string(password); pswd != expectedPassword {
				return nil, fmt.Errorf("incorrect password %q (expected %q)", pswd, expectedPassword)
			}
			return &ssh.Permissions{}, nil
		},
	}
	serverConfig.AddHostKey(hostkey)

	listener, listenErr := net.Listen("tcp", "localhost:0")
	require.NoError(tb, listenErr)
	tb.Cleanup(func() { listener.Close() })

	go func() {
		for {
			tcpConn, acptErr := listener.Accept()
			if errors.Is(acptErr, net.ErrClosed) {
				return
			}
			require.NoError(tb, acptErr)

			go func() {
				defer tcpConn.Close()

				sshConn, channels, reqs, sshErr := ssh.NewServerConn(tcpConn, serverConfig)
				require.NoError(tb, sshErr)
				defer sshConn.Close()
				go ssh.DiscardRequests(reqs)

				for newChannel := range channels {
					go func() {
						if newChannel.ChannelType() != "session" {
							require.NoError(tb, newChannel.Reject(ssh.UnknownChannelType, "unknown channel type"))
						}

						sftpSession, requests, channelOpenErr := newChannel.Accept()
						require.NoError(tb, channelOpenErr)
						defer sftpSession.Close()
						go acceptRequests(tb, requests)

						handler := sftplib.Handlers{
							FileList: testSFTPLister{tb},
						}

						sftpServer := sftplib.NewRequestServer(sftpSession, handler)
						require.ErrorIs(tb, sftpServer.Serve(), io.EOF)
					}()
				}
			}()
		}
	}()

	return &testModels{
		partner: &model.RemoteAgent{
			Name:     "sftp_partner",
			Protocol: sftp.SFTP,
			Address:  types.MustAddr(listener.Addr().String()),
		},
		partnerCredentials: []*model.Credential{{
			Name:  "hostkey",
			Type:  sftp.AuthSSHPublicKey,
			Value: hostkeyPBK,
		}},
		user: &model.RemoteAccount{
			Login: expectedUsername,
		},
		userCredentials: []*model.Credential{{
			Name:  "password",
			Type:  auth.Password,
			Value: expectedPassword,
		}},
	}
}

type testSFTPLister struct{ testing.TB }

func (t testSFTPLister) Filelist(req *sftplib.Request) (sftplib.ListerAt, error) {
	dir, err := memFs.Open(validPath(req.Filepath))
	if err != nil {
		return nil, err
	}

	return testSFTPListerAt{dir: dir, TB: t.TB}, nil
}

type testSFTPListerAt struct {
	dir fs.File
	testing.TB
}

func (t testSFTPListerAt) ListAt(infos []os.FileInfo, offset int64) (int, error) {
	results, err := readDirInfoFile(t.dir, -1)
	if err != nil {
		return 0, err
	}

	if len(results) == 0 {
		return 0, io.EOF
	}

	n := copy(infos, results[offset:])
	if n <= len(results) {
		return n, io.EOF
	}

	return n, nil
}

func acceptRequests(tb testing.TB, in <-chan *ssh.Request) {
	for req := range in {
		ok := req.Type == "subsystem" && string(req.Payload[4:]) == "sftp"

		require.NoError(tb, req.Reply(ok, nil))
	}
}
