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
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"github.com/pkg/sftp"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/ssh"
)

func init() {
	conf := &ssh.ServerConfig{
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

	conf.AddHostKey(privateKey)

	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		log.Fatalf("Failed to open SFTP server key: %s", err)
	}
	clientTestPort = uint16(listener.Addr().(*net.TCPAddr).Port)

	go handleSFTP(listener, conf)
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
	Convey("Given a SFTP client", t, func() {
		client := &Client{}

		Convey("Given a valid address", func() {
			client.conf = &config.SftpProtoConfig{
				Port:    clientTestPort,
				Address: "localhost",
			}

			Convey("When calling the `Connect` method", func() {
				err := client.Connect()

				Convey("Then it should NOT return an error", func() {
					So(err, ShouldBeNil)

					Convey("Then the connection should be open", func() {
						So(client.conn, ShouldNotBeNil)
						So(client.conn.Close(), ShouldBeNil)
					})
				})
			})
		})

		Convey("Given an incorrect address", func() {
			client.conf = &config.SftpProtoConfig{
				Port:    clientTestPort,
				Address: "255.255.255.255",
			}

			Convey("When calling the `Connect` method", func() {
				err := client.Connect()

				Convey("Then it should return an error", func() {
					So(err, ShouldBeError)

					Convey("Then the connection should NOT be open", func() {
						So(client.conn, ShouldBeNil)
					})
				})
			})
		})
	})
}

func TestAuthenticate(t *testing.T) {
	Convey("Given a SFTP client", t, func() {
		client := &Client{
			conf: &config.SftpProtoConfig{
				Port:    clientTestPort,
				Address: "localhost",
			},
		}
		So(client.Connect(), ShouldBeNil)
		Reset(func() { _ = client.conn.Close() })

		Convey("Given a valid SFTP configuration", func() {
			client.Info = model.OutTransferInfo{
				Account: model.RemoteAccount{
					Login:    testLogin,
					Password: []byte(testPassword),
				},
				Certs: []model.Cert{{
					PublicKey: testPBK,
				}},
			}

			err := client.Authenticate()

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)

				Convey("Then the SSH tunnel should be opened", func() {
					So(client.client, ShouldNotBeNil)
					So(client.client.Close(), ShouldBeNil)
				})
			})
		})

		SkipConvey("Given an incorrect SFTP configuration", func() {
			client.Info = model.OutTransferInfo{
				Account: model.RemoteAccount{
					Login:    testLogin,
					Password: []byte("tutu"),
				},
				Certs: []model.Cert{{
					PublicKey: testPBK,
				}},
			}

			err := client.Authenticate()

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)

				Convey("Then the SSH tunnel NOT should be opened", func() {
					So(client.client, ShouldBeNil)
				})
			})
		})
	})
}

func TestRequest(t *testing.T) {
	Convey("Given a SFTP client", t, func() {
		client := &Client{
			conf: &config.SftpProtoConfig{
				Port:    clientTestPort,
				Address: "localhost",
			},
		}
		So(client.Connect(), ShouldBeNil)
		Reset(func() { _ = client.conn.Close() })

		client.Info = model.OutTransferInfo{
			Account: model.RemoteAccount{
				Login:    testLogin,
				Password: []byte(testPassword),
			},
			Certs: []model.Cert{{
				PublicKey: testPBK,
			}},
		}

		So(client.Authenticate(), ShouldBeNil)
		Reset(func() { _ = client.client.Close() })

		Convey("Given a valid out file transfer", func() {
			client.Info.Transfer = model.Transfer{
				DestPath: "client_test.dst",
			}
			client.Info.Rule = model.Rule{
				IsSend: true,
			}

			err := client.Request()
			Reset(func() { _ = os.Remove(client.Info.Transfer.DestPath) })

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)

				Convey("Then the file stream should be open", func() {
					So(client.remoteFile, ShouldNotBeNil)
					So(client.remoteFile.Close(), ShouldBeNil)
				})
			})
		})

		Convey("Given a valid in file transfer", func() {
			client.Info.Transfer = model.Transfer{
				SourcePath: "client.go",
			}
			client.Info.Rule = model.Rule{
				IsSend: false,
			}

			err := client.Request()

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)

				Convey("Then the file stream should be open", func() {
					So(client.remoteFile, ShouldNotBeNil)
					So(client.remoteFile.Close(), ShouldBeNil)
				})
			})
		})

		Convey("Given an invalid in file transfer", func() {
			client.Info.Transfer = model.Transfer{
				SourcePath: "unknown.file",
			}
			client.Info.Rule = model.Rule{
				IsSend: true,
			}

			err := client.Request()

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)

				Convey("Then the file stream should NOT be open", func() {
					So(client.remoteFile, ShouldBeNil)
				})
			})
		})
	})
}

func TestData(t *testing.T) {
	Convey("Given a SFTP client", t, func() {
		client := &Client{
			conf: &config.SftpProtoConfig{
				Port:    clientTestPort,
				Address: "localhost",
			},
		}
		So(client.Connect(), ShouldBeNil)
		Reset(func() { _ = client.conn.Close() })

		client.Info = model.OutTransferInfo{
			Account: model.RemoteAccount{
				Login:    testLogin,
				Password: []byte(testPassword),
			},
			Certs: []model.Cert{{
				PublicKey: testPBK,
			}},
		}

		So(client.Authenticate(), ShouldBeNil)
		Reset(func() { _ = client.client.Close() })

		Convey("Given a valid out file transfer", func() {
			client.Info.Transfer = model.Transfer{
				DestPath:   "client_test_in.dst",
				SourcePath: "client.go",
			}
			client.Info.Rule = model.Rule{
				IsSend: true,
			}

			So(client.Request(), ShouldBeNil)
			Reset(func() { _ = os.Remove(client.Info.Transfer.DestPath) })

			file, err := os.Open(client.Info.Transfer.SourcePath)
			So(err, ShouldBeNil)
			client.localFile = file

			err = client.Data()

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)

				Convey("Then the file should have been copied", func() {
					src, err := ioutil.ReadFile(client.Info.Transfer.SourcePath)
					So(err, ShouldBeNil)
					dst, err := ioutil.ReadFile(client.Info.Transfer.DestPath)
					So(err, ShouldBeNil)

					So(src, ShouldResemble, dst)
				})
			})
		})

		Convey("Given a valid in file transfer", func() {
			client.Info.Transfer = model.Transfer{
				DestPath:   "client_test_out.dst",
				SourcePath: "client.go",
			}
			client.Info.Rule = model.Rule{
				IsSend: false,
			}

			So(client.Request(), ShouldBeNil)

			file, err := os.Create(client.Info.Transfer.DestPath)
			So(err, ShouldBeNil)
			client.localFile = file
			Reset(func() { _ = os.Remove(client.Info.Transfer.DestPath) })

			err = client.Data()

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)

				Convey("Then the file should have been copied", func() {
					src, err := ioutil.ReadFile(client.Info.Transfer.SourcePath)
					So(err, ShouldBeNil)
					dst, err := ioutil.ReadFile(client.Info.Transfer.DestPath)
					So(err, ShouldBeNil)

					So(src, ShouldResemble, dst)
				})
			})
		})
	})
}
