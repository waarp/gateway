package sftp

import (
	. "github.com/smartystreets/goconvey/convey"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func makeDummyClient(addr, login, pwd string) *sftp.Client {
	key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(rsaPBK)) //nolint:dogsled
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

	return cli
}
