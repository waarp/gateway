package sftp

import (
	"github.com/pkg/sftp"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/ssh"
)

type dummyClient struct {
	*sftp.Client
	conn *ssh.Client
}

func makeDummyClient(addr, login, pwd string) *dummyClient {
	//nolint:dogsled // this is caused by the design of a third party library
	key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(rsaPBK))
	So(err, ShouldBeNil)

	clientConf := &ssh.ClientConfig{
		User: login,
		Auth: []ssh.AuthMethod{
			ssh.Password(pwd),
		},
		HostKeyCallback: ssh.FixedHostKey(key),
	}

	conn, err := ssh.Dial("tcp", addr, clientConf)
	So(err, ShouldBeNil)
	Reset(func() { _ = conn.Close() })

	cli, err := sftp.NewClient(conn)
	So(err, ShouldBeNil)

	return &dummyClient{Client: cli, conn: conn}
}
