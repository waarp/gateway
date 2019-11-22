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

const (
	testSFTPPubKey   = "test_sftp_root/id_rsa.pub"
	testSFTPPrivKey  = "test_sftp_root/id_rsa"
	testInitPort     = 0
	testSFTPUser     = "test_user"
	testSFTPPassword = "test_password"
)

var testSFTPPort int

func init() {
	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if c.User() == testSFTPUser && string(pass) == testSFTPPassword {
				return nil, nil
			}
			return nil, fmt.Errorf("password '%s' rejected for user '%s'", pass, c.User())
		},
	}

	privateBytes, err := ioutil.ReadFile(testSFTPPrivKey)
	if err != nil {
		log.Fatal("Failed to open SFTP server key", err)
	}

	privateKey, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		log.Fatal("Failed to parse SFTP server key", err)
	}

	config.AddHostKey(privateKey)

	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%v", testInitPort))
	if err != nil {
		log.Fatal("Failed to open SFTP server key", err)
	}
	testSFTPPort = listener.Addr().(*net.TCPAddr).Port

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
				fmt.Sprintf(`{"address":%s, "port":%d}`, `"127.0.0.1"`, testSFTPPort)),
		}

		Convey("Given a valid certificate", func() {
			serverKey, err := ioutil.ReadFile(testSFTPPubKey)
			if err != nil {
				t.Fatal(err)
			}
			cert := &model.Cert{
				PublicKey: serverKey,
			}

			Convey("Given a valid Remote Account", func() {
				account := &model.RemoteAccount{
					Login:    testSFTPUser,
					Password: []byte(testSFTPPassword),
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
					Login:    testSFTPUser,
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
					Login:    testSFTPUser,
					Password: []byte(testSFTPPassword),
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

func TestClose(t *testing.T) {
	Convey("Given a SFTP Server registered as a Remote Agent", t, func() {
		remote := &model.RemoteAgent{
			Name:     "test",
			Protocol: "sftp",
			ProtoConfig: []byte(
				fmt.Sprintf(`{"address":%s, "port":%d}`, `"127.0.0.1"`, testSFTPPort)),
		}

		Convey("Given a valid connection to the server", func() {
			serverKey, err := ioutil.ReadFile(testSFTPPubKey)
			if err != nil {
				t.Fatal(err)
			}
			cert := &model.Cert{
				PublicKey: serverKey,
			}
			account := &model.RemoteAccount{
				Login:    testSFTPUser,
				Password: []byte(testSFTPPassword),
			}
			context, err := Connect(remote, cert, account)
			if err != nil {
				t.Fatal(err)
			}
			if context != nil {
				defer context.Close()
			}

			Convey("When calling the Close function", func() {
				context.Close()

				Convey("Then when trying to use thhe ssh connection", func() {
					err := context.SSHClient.Wait()

					Convey("Then err should NOT be nil", func() {
						So(err, ShouldNotBeNil)
					})

					Convey("err should be Used of closed connection", func() {
						So(err.Error(), ShouldContainSubstring, "use of closed network connection")
					})
				})
			})
		})
	})
}

func TestDoTransfer(t *testing.T) {
	Convey("Given a SFTP Server registered as a Remote Agent", t, func() {
		remote := &model.RemoteAgent{
			Name:     "test",
			Protocol: "sftp",
			ProtoConfig: []byte(
				fmt.Sprintf(`{"address":%s, "port":%d}`, `"127.0.0.1"`, testSFTPPort)),
		}

		Convey("Given a valid connection to the server", func() {
			serverKey, err := ioutil.ReadFile(testSFTPPubKey)
			if err != nil {
				t.Fatal(err)
			}
			cert := &model.Cert{
				PublicKey: serverKey,
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
				Login:    testSFTPUser,
				Password: []byte(testSFTPPassword),
			}
			context, err := Connect(remote, cert, account)
			if err != nil {
				t.Fatal(err)
			}
			if context != nil {
				defer context.Close()
			}

			// TODO Handle transfer rules
			Convey("Given a valid push transfer", func() {
				transfer := &model.Transfer{
					RuleID:     push.ID,
					RemoteID:   remote.ID,
					AccountID:  account.ID,
					SourcePath: "client.go",
					DestPath:   "test_sftp_root/client.ds",
				}

				Convey("When calling DoTransfer", func() {
					err := DoTransfer(context.SftpClient, transfer, push)

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
					DestPath:   "test_sftp_root/client.ds",
				}

				Convey("When calling DoTransfer", func() {
					err := DoTransfer(context.SftpClient, transfer, push)

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
				transfer := &model.Transfer{
					RuleID:     pull.ID,
					RemoteID:   remote.ID,
					AccountID:  account.ID,
					SourcePath: "test_sftp_root/test.src",
					DestPath:   "test.ds",
				}

				Convey("When calling DoTransfer", func() {

					err := DoTransfer(context.SftpClient, transfer, pull)

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
