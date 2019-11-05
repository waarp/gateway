package sftp

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/pkg/sftp"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/ssh"
)

func init() {
	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if c.User() == testLogin && string(pass) == testPassword {
				return nil, nil
			}
			return nil, fmt.Errorf("password '%s' rejected for user '%s'", pass, c.User())
		},
	}

	privateKey, err := ssh.ParsePrivateKey(testPK)
	if err != nil {
		log.Fatalf("Failed to parse SFTP server key: %s", err)
	}

	config.AddHostKey(privateKey)

	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		log.Fatalf("Failed to open SFTP server key: %s", err)
	}
	port = listener.Addr().(*net.TCPAddr).Port

	go handleSFTP(listener, config)
}

func handleSFTP(listener net.Listener, config *ssh.ServerConfig) {
	for {
		nConn, err := listener.Accept()
		if err != nil {
			log.Println("Failed to accept incoming connection", err)
			continue
		}

		_, chans, reqs, err := ssh.NewServerConn(nConn, config)
		if err != nil {
			log.Println("Failed to handshake", err)
			continue
		}

		go ssh.DiscardRequests(reqs)

		for newChannel := range chans {
			if newChannel.ChannelType() != "session" {
				_ = newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
				continue
			}
			channel, requests, err := newChannel.Accept()
			if err != nil {
				log.Println("Could not accept channel.", err)
				break
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
					_ = req.Reply(ok, nil)
				}
			}(requests)

			server, err := sftp.NewServer(channel)
			if err != nil {
				log.Println("Failed to start SFTP server", err)
				break
			}
			if err := server.Serve(); err == io.EOF {
				_ = server.Close()
				break
			} else if err != nil {
				log.Println("sftp server completed with error:", err)
				break
			}
		}
	}
}

func TestConnect(t *testing.T) {
	Convey("Given a SFTP Server registered as a Remote Agent", t, func() {
		remote := &model.RemoteAgent{
			Name:     "test",
			Protocol: "sftp",
			ProtoConfig: []byte(
				fmt.Sprintf(`{"address":%s, "port":%d}`, `"127.0.0.1"`, port)),
		}

		Convey("Given a valid certificate", func() {
			cert := &model.Cert{
				PublicKey: testPBK,
			}

			Convey("Given a valid Remote Account", func() {
				account := &model.RemoteAccount{
					Login:    testLogin,
					Password: []byte(testPassword),
				}

				Convey("When calling 'Connect' function", func() {
					client, err := Connect(remote, cert, account)
					Convey("Then it should return a client", func() {
						So(client, ShouldNotBeNil)
					})

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})
					if client != nil {
						client.Close()
					}
				})
			})
			Convey("Given an incorrect Account", func() {
				account := &model.RemoteAccount{
					Login:    testLogin,
					Password: []byte("Giberish"),
				}
				Convey("When calling 'Connect' function", func() {
					client, err := Connect(remote, cert, account)

					Convey("Then it should return NO client", func() {
						So(client, ShouldBeNil)
					})

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
					})
				})
			})
		})
		Convey("Given a unvalid certificate", func() {
			cert := &model.Cert{
				PublicKey: []byte("Giberish"),
			}
			Convey("Given a valid Remote Account", func() {
				account := &model.RemoteAccount{
					Login:    testLogin,
					Password: []byte(testPassword),
				}

				Convey("When calling 'Connect' function", func() {
					client, err := Connect(remote, cert, account)

					Convey("Then it should return NO client", func() {
						So(client, ShouldBeNil)
					})

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
					})
				})
			})
		})
	})

}

func TestDoTransfer(t *testing.T) {
	Convey("Given a SFTP Server registered as a Remote Agent", t, func() {
		root := "client_test_root"
		So(os.Mkdir(root, 0700), ShouldBeNil)
		Reset(func() { _ = os.RemoveAll(root) })

		remote := &model.RemoteAgent{
			Name:     "test",
			Protocol: "sftp",
			ProtoConfig: []byte(
				fmt.Sprintf(`{"address":%s, "port":%d}`, `"127.0.0.1"`, port)),
		}

		Convey("Given a valid connection to the server", func() {
			cert := &model.Cert{
				PublicKey: testPBK,
			}
			push := &model.Rule{
				Name:  "push",
				IsGet: false,
			}
			pull := &model.Rule{
				Name:  "pull",
				IsGet: true,
			}
			account := &model.RemoteAccount{
				Login:    testLogin,
				Password: []byte(testPassword),
			}
			client, err := Connect(remote, cert, account)
			if err != nil {
				t.Fatal(err)
			}
			if client != nil {
				defer client.Close()
			}

			// TODO Handle transfer rules
			Convey("Given a valid push transfer", func() {

				transfer := &model.Transfer{
					RuleID:     push.ID,
					RemoteID:   remote.ID,
					AccountID:  account.ID,
					SourcePath: "client.go",
					DestPath:   root + "/client.ds",
				}

				Convey("When calling DoTransfer", func() {
					err := DoTransfer(client, transfer, push)

					Reset(func() {
						_ = os.Remove(transfer.DestPath)
					})

					Convey("It should return no error", func() {
						So(err, ShouldBeNil)
					})

					Convey("The destination file should exist", func() {
						dstContent, err := ioutil.ReadFile(transfer.DestPath)
						So(err, ShouldBeNil)

						Convey("The destination file should be the same as the source file", func() {
							srcContent, err := ioutil.ReadFile(transfer.SourcePath)
							So(err, ShouldBeNil)

							So(dstContent, ShouldResemble, srcContent)
						})
					})
				})
			})

			Convey("Given a push transfer with a non exiting file", func() {
				transfer := &model.Transfer{
					RuleID:     push.ID,
					RemoteID:   remote.ID,
					AccountID:  account.ID,
					SourcePath: "unknown",
					DestPath:   root + "/client.ds",
				}

				Convey("When calling DoTransfer", func() {
					err := DoTransfer(client, transfer, push)

					Reset(func() {
						_ = os.Remove(transfer.DestPath)
					})

					Convey("It should return an error", func() {
						So(err, ShouldBeError)
					})

					Convey("The destination file should NOT exist", func() {
						_, err := os.Open(transfer.DestPath)
						So(err, ShouldBeError)
						So(err, ShouldHaveSameTypeAs, &os.PathError{})
					})
				})
			})

			Convey("Given a valid pull transfer", func() {
				err := ioutil.WriteFile(root+"/test_pull.src",
					[]byte("test client pull"), 0600)
				So(err, ShouldBeNil)

				transfer := &model.Transfer{
					RuleID:     pull.ID,
					RemoteID:   remote.ID,
					AccountID:  account.ID,
					SourcePath: root + "/test_pull.src",
					DestPath:   "test_pull.dst",
				}

				Convey("When calling DoTransfer", func() {

					err := DoTransfer(client, transfer, pull)

					Reset(func() {
						_ = os.Remove(transfer.DestPath)
					})

					Convey("It should return no error", func() {
						So(err, ShouldBeNil)
					})

					Convey("The destination file should exist", func() {
						dstContent, err := ioutil.ReadFile(transfer.DestPath)
						So(err, ShouldBeNil)

						Convey("The destination file should be the same as the source file", func() {
							srcContent, err := ioutil.ReadFile(transfer.SourcePath)
							So(err, ShouldBeNil)

							So(dstContent, ShouldResemble, srcContent)
						})
					})
				})
			})
		})
	})
}
