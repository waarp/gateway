package sftp

import (
	"bytes"
	"context"
	"crypto/rand"
	"io"
	"io/ioutil"
	"net"
	"os"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/pkg/sftp"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/ssh"
)

func TestServerStop(t *testing.T) {
	port := "2021"

	Convey("Given a running SFTP server service", t, func() {
		db := database.GetTestDatabase()
		agent := &model.LocalAgent{
			Name:     "test_sftp_server",
			Protocol: "sftp",
			ProtoConfig: []byte(`{"address":"localhost","port":` + port +
				`, "root":"root"}`),
		}
		So(db.Create(agent), ShouldBeNil)

		cert := &model.Cert{
			OwnerType:   agent.TableName(),
			OwnerID:     agent.ID,
			Name:        "test_sftp_server_cert",
			PrivateKey:  testPK,
			PublicKey:   testPBK,
			Certificate: []byte("cert"),
		}
		So(db.Create(cert), ShouldBeNil)

		server := NewServer(db, agent, log.NewLogger("test_sftp_server", testLogConf))
		So(server.Start(), ShouldBeNil)

		Convey("When stopping the service", func() {
			err := server.Stop(context.Background())

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)

				Convey("Then the SFTP server should no longer respond", func() {
					_, err := ssh.Dial("tcp", "localhost:"+port, &ssh.ClientConfig{})
					So(err, ShouldBeError)
				})
			})
		})
	})
}

func TestServerStart(t *testing.T) {
	port := "2022"

	Convey("Given an SFTP server service", t, func() {
		db := database.GetTestDatabase()

		agent := &model.LocalAgent{
			Name:     "test_sftp_server",
			Protocol: "sftp",
			ProtoConfig: []byte(`{"address":"localhost","port":` + port +
				`,"root":"root"}`),
		}
		So(db.Create(agent), ShouldBeNil)

		cert := &model.Cert{
			OwnerType:   agent.TableName(),
			OwnerID:     agent.ID,
			Name:        "test_sftp_server_cert",
			PrivateKey:  testPK,
			PublicKey:   testPBK,
			Certificate: []byte("cert"),
		}
		So(db.Create(cert), ShouldBeNil)

		sftpServer := NewServer(db, agent, log.NewLogger("test_sftp_server", testLogConf))

		Convey("When starting the server", func() {
			err := sftpServer.Start()

			Reset(func() {
				_ = sftpServer.Stop(context.Background())
			})

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestSSHServer(t *testing.T) {
	port := "2023"
	logger := log.NewLogger("test_sftp_server", testLogConf)

	Convey("Given an SFTP server", t, func() {
		root, err := ioutil.TempDir("", "gateway-test")
		So(err, ShouldBeNil)
		Reset(func() { _ = os.RemoveAll(root) })

		db := database.GetTestDatabase()
		agent := &model.LocalAgent{
			Name:     "test_sftp_server",
			Protocol: "sftp",
			ProtoConfig: []byte(`{"address":"localhost","port":` + port + `,"root":"` +
				root + `"}`),
		}
		So(db.Create(agent), ShouldBeNil)

		pwd := "tata"
		user := &model.LocalAccount{
			LocalAgentID: agent.ID,
			Login:        "toto",
			Password:     []byte(pwd),
		}
		So(db.Create(user), ShouldBeNil)

		cert := &model.Cert{
			OwnerType:   agent.TableName(),
			OwnerID:     agent.ID,
			Name:        "test_sftp_server_cert",
			PrivateKey:  testPK,
			PublicKey:   testPBK,
			Certificate: []byte("cert"),
		}
		So(db.Create(cert), ShouldBeNil)

		pull := &model.Rule{
			Name:    "pull",
			Comment: "",
			IsSend:  false,
			Path:    "/pull/in",
		}
		push := &model.Rule{
			Name:    "push",
			Comment: "",
			IsSend:  true,
			Path:    "/push/out",
		}
		So(db.Create(pull), ShouldBeNil)
		So(db.Create(push), ShouldBeNil)

		config, err := loadSSHConfig(db, cert)
		So(err, ShouldBeNil)

		listener, err := net.Listen("tcp", "localhost:"+port)
		So(err, ShouldBeNil)

		sshServ := newListener(db, logger, listener, agent, config)
		sshServ.listen()

		Reset(func() {
			select {
			case <-sshServ.shutdown:
			default:
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				So(sshServ.close(ctx), ShouldBeNil)
			}
		})

		Convey("Given an SSH client", func() {
			key, _, _, _, err := ssh.ParseAuthorizedKey(testPBK) //nolint:dogsled
			So(err, ShouldBeNil)

			clientConf := &ssh.ClientConfig{
				User: user.Login,
				Auth: []ssh.AuthMethod{
					ssh.Password(pwd),
				},
				HostKeyCallback: ssh.FixedHostKey(key),
			}

			conn, err := ssh.Dial("tcp", "localhost:"+port, clientConf)
			So(err, ShouldBeNil)
			Reset(func() { _ = conn.Close() })

			client, err := sftp.NewClient(conn)
			So(err, ShouldBeNil)

			Convey("Given an incoming transfer", func() {
				content := "Test incoming file"

				err := os.MkdirAll(root+"/pull/in", 0700)
				So(err, ShouldBeNil)

				Reset(func() { _ = os.RemoveAll(root + "/pull") })

				Convey("Given that the transfer finishes normally", func() {
					src := bytes.NewBuffer([]byte(content))

					dst, err := client.Create("/pull/in/test_in.dst")
					So(err, ShouldBeNil)

					_, err = io.Copy(dst, src)
					So(err, ShouldBeNil)

					err = dst.Close()
					So(err, ShouldBeNil)

					Convey("Then the destination file should exist", func() {
						path := root + "/pull/in/test_in.dst"
						_, err := os.Stat(path)
						So(err, ShouldBeNil)

						Convey("Then the file's content should be identical to the original", func() {
							dstContent, err := ioutil.ReadFile(path)
							So(err, ShouldBeNil)

							So(string(dstContent), ShouldEqual, content)
						})
					})

					Convey("Then the transfer should appear in the history", func() {
						hist := &model.TransferHistory{
							IsServer:       true,
							IsSend:         false,
							Account:        user.Login,
							Remote:         agent.Name,
							Protocol:       "sftp",
							SourceFilename: "test_in.dst",
							DestFilename:   "/pull/in/test_in.dst",
							Rule:           "pull",
							Status:         model.StatusDone,
						}

						err := client.Close()
						So(err, ShouldBeNil)

						ok, err := db.Exists(hist)
						So(err, ShouldBeNil)
						So(ok, ShouldBeTrue)
					})
				})

				Convey("Given that 2 transfers launch simultaneously", func() {
					src := bytes.NewBuffer([]byte(content))

					dst1, err := client.Create("/pull/in/test_in_1.dst")
					So(err, ShouldBeNil)
					dst2, err := client.Create("/pull/in/test_in_2.dst")
					So(err, ShouldBeNil)

					res1 := make(chan error)
					res2 := make(chan error)
					go func() {
						_, err := io.Copy(dst1, src)
						res1 <- err
					}()
					go func() {
						_, err := io.Copy(dst2, src)
						res2 <- err
					}()

					So(<-res1, ShouldBeNil)
					So(<-res2, ShouldBeNil)

					err = dst1.Close()
					So(err, ShouldBeNil)
					err = dst2.Close()
					So(err, ShouldBeNil)

					Convey("Then the destination files should exist", func() {
						path1 := root + "/pull/in/test_in_1.dst"
						_, err := os.Stat(path1)
						So(err, ShouldBeNil)

						path2 := root + "/pull/in/test_in_2.dst"
						_, err = os.Stat(path2)
						So(err, ShouldBeNil)

						Convey("Then the files' content should be identical to "+
							"the originals", func() {
							dstContent1, err := ioutil.ReadFile(path1)
							So(err, ShouldBeNil)
							dstContent2, err := ioutil.ReadFile(path2)
							So(err, ShouldBeNil)

							So(string(dstContent1), ShouldEqual, content)
							So(string(dstContent2), ShouldEqual, content)
						})
					})

					Convey("Then the transfers should appear in the history", func() {
						hist1 := &model.TransferHistory{
							IsServer:       true,
							IsSend:         false,
							Account:        user.Login,
							Remote:         agent.Name,
							Protocol:       "sftp",
							SourceFilename: "test_in_1.dst",
							DestFilename:   "/pull/in/test_in_1.dst",
							Rule:           "pull",
							Status:         model.StatusDone,
						}
						hist2 := &model.TransferHistory{
							IsServer:       true,
							IsSend:         false,
							Account:        user.Login,
							Remote:         agent.Name,
							Protocol:       "sftp",
							SourceFilename: "test_in_2.dst",
							DestFilename:   "/pull/in/test_in_2.dst",
							Rule:           "pull",
							Status:         model.StatusDone,
						}

						err := client.Close()
						So(err, ShouldBeNil)

						ok, err := db.Exists(hist1)
						So(err, ShouldBeNil)
						So(ok, ShouldBeTrue)

						ok, err = db.Exists(hist2)
						So(err, ShouldBeNil)
						So(ok, ShouldBeTrue)
					})
				})

				Convey("Given that the transfer fails", func() {
					src := rand.Reader

					dst, err := client.Create("/pull/in/test_in_fail.dst")
					So(err, ShouldBeNil)

					reply := make(chan error)
					go func() {
						_, res := io.Copy(dst, src)
						reply <- res
					}()

					err = client.Close()
					So(err, ShouldBeNil)
					err = <-reply

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "failed to send packet: EOF")
					})

					Convey("Then the transfer should appear in the history", func() {
						hist := &model.TransferHistory{
							IsServer:       true,
							IsSend:         false,
							Account:        user.Login,
							Remote:         agent.Name,
							Protocol:       "sftp",
							SourceFilename: "test_in_fail.dst",
							DestFilename:   "/pull/in/test_in_fail.dst",
							Rule:           "pull",
							Status:         model.StatusError,
						}

						ok, err := db.Exists(hist)
						So(err, ShouldBeNil)
						So(ok, ShouldBeTrue)
					})
				})

				Convey("Given that the server shuts down", func() {
					src := rand.Reader

					dst, err := client.Create("/pull/in/test_in_shutdown.dst")
					So(err, ShouldBeNil)

					go func() {
						_, _ = io.Copy(dst, src)
					}()

					ctx, cancel := context.WithTimeout(context.Background(), time.Second)
					defer cancel()
					So(sshServ.close(ctx), ShouldBeNil)

					Convey("Then the transfer should appear in the history", func() {
						hist := &model.TransferHistory{
							IsServer:       true,
							IsSend:         false,
							Account:        user.Login,
							Remote:         agent.Name,
							Protocol:       "sftp",
							SourceFilename: "test_in_shutdown.dst",
							DestFilename:   "/pull/in/test_in_shutdown.dst",
							Rule:           "pull",
							Status:         model.StatusError,
						}

						ok, err := db.Exists(hist)
						So(err, ShouldBeNil)
						So(ok, ShouldBeTrue)
					})
				})
			})

			Convey("Given an outgoing transfer", func() {
				content := "Test outgoing file"

				err := os.MkdirAll(root+"/push/out", 0700)
				So(err, ShouldBeNil)

				err = ioutil.WriteFile(root+"/push/out/test_out.src", []byte(content), 0600)
				So(err, ShouldBeNil)

				Reset(func() { _ = os.RemoveAll(root + "/push") })

				Convey("Given that the transfer finishes normally", func() {
					src, err := client.Open("/push/out/test_out.src")
					So(err, ShouldBeNil)

					dst := &bytes.Buffer{}

					_, err = src.WriteTo(dst)
					So(err, ShouldBeNil)

					err = src.Close()
					So(err, ShouldBeNil)

					Convey("Then the destination file should exist", func() {
						dstContent, err := ioutil.ReadAll(dst)
						So(err, ShouldBeNil)
						So(dstContent, ShouldNotBeNil)

						Convey("Then the file's content should be identical to the original", func() {
							So(string(dstContent), ShouldEqual, content)
						})
					})

					Convey("Then the transfer should appear in the history", func() {
						hist := &model.TransferHistory{
							IsServer:       true,
							IsSend:         true,
							Account:        user.Login,
							Remote:         agent.Name,
							Protocol:       "sftp",
							SourceFilename: "/push/out/test_out.src",
							DestFilename:   "test_out.src",
							Rule:           "push",
							Status:         model.StatusDone,
						}

						err := client.Close()
						So(err, ShouldBeNil)

						ok, err := db.Exists(hist)
						So(err, ShouldBeNil)
						So(ok, ShouldBeTrue)
					})
				})
			})
		})
	})
}
