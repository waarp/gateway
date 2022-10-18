package sftp

import (
	"github.com/pkg/sftp"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/ssh"

	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

type dummyClient struct {
	*sftp.Client
	conn *ssh.Client
}

func makeDummyClient(addr, login, pwd string) *dummyClient {
	key, err := utils.ParseSSHAuthorizedKey(rsaPBK)
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
