package sftp

import (
	"io"
	"log"
	"net"
	"os"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func makeDummyClient(addr, login, pwd string) *sftp.Client {
	key, _, _, _, err := ssh.ParseAuthorizedKey(testPBK) //nolint:dogsled
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

	cli, err := sftp.NewClient(conn)
	So(err, ShouldBeNil)

	return cli
}

func makeDummyServer(root string, addr string) {
	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			return nil, nil
		},
	}

	private, err := ssh.ParsePrivateKey([]byte(rsaPK))
	So(err, ShouldBeNil)
	config.AddHostKey(private)

	listener, err := net.Listen("tcp", addr)
	So(err, ShouldBeNil)

	old, err := os.Getwd()
	So(err, ShouldBeNil)
	So(os.Chdir(root), ShouldBeNil)
	Reset(func() { So(os.Chdir(old), ShouldBeNil) })

	go func() {
		nConn, err := listener.Accept()
		if err != nil {
			log.Fatal("failed to accept incoming connection", err)
		}

		_, chans, reqs, err := ssh.NewServerConn(nConn, config)
		if err != nil {
			log.Fatal("failed to handshake", err)
		}

		go ssh.DiscardRequests(reqs)

		for newChannel := range chans {
			if newChannel.ChannelType() != "session" {
				newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
				continue
			}
			channel, requests, err := newChannel.Accept()
			if err != nil {
				log.Fatal("could not accept channel.", err)
			}

			go func(in <-chan *ssh.Request) {
				for req := range in {
					ok := false
					switch req.Type {
					case "subsystem":
						if string(req.Payload[4:]) == "sftp" {
							ok = true
						}
					}
					req.Reply(ok, nil)
				}
			}(requests)

			server, err := sftp.NewServer(channel)
			if err != nil {
				continue
			}
			if err := server.Serve(); err == io.EOF {
				_ = server.Close()
			} else if err != nil {
				log.Fatal("sftp server completed with error:", err)
			}
		}
	}()
}
