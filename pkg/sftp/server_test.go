package sftp

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net"
	"os"
	"path"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/pkg/sftp"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/ssh"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
)

func getTestPort() string {
	listener, err := net.Listen("tcp", "localhost:0")
	So(err, ShouldBeNil)
	_, port, err := net.SplitHostPort(listener.Addr().String())
	So(err, ShouldBeNil)
	So(listener.Close(), ShouldBeNil)

	return port
}

func TestServerStop(t *testing.T) {

	Convey("Given a running SFTP server service", t, func() {
		db := database.GetTestDatabase()
		port := getTestPort()

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

		server := NewService(db, agent, log.NewLogger("test_sftp_server"))
		So(server.Start(), ShouldBeNil)

		Convey("When stopping the service", func() {
			err := server.Stop(context.Background())

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)

				Convey("Then the SFTP server should no longer respond", func() {
					_, err := ssh.Dial("tcp", "localhost:"+port, &ssh.ClientConfig{})
					So(err, ShouldNotBeNil)
				})
			})
		})
	})
}

func TestServerStart(t *testing.T) {
	Convey("Given an SFTP server service", t, func() {
		db := database.GetTestDatabase()
		port := getTestPort()
		root, err := filepath.Abs("server_start_root")
		So(err, ShouldBeNil)

		agent := &model.LocalAgent{
			Name:        "test_sftp_server",
			Protocol:    "sftp",
			Paths:       &model.ServerPaths{Root: root},
			ProtoConfig: []byte(`{"address":"localhost","port":` + port + `}`),
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

		sftpServer := NewService(db, agent, log.NewLogger("test_sftp_server"))

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
	logger := log.NewLogger("test_sftp_server")

	Convey("Given a server root", t, func() {
		root, err := filepath.Abs("test_server_root")
		So(err, ShouldBeNil)
		So(os.Mkdir(root, 0700), ShouldBeNil)
		Reset(func() { _ = os.RemoveAll(root) })

		Convey("Given an SFTP server", func() {
			listener, err := net.Listen("tcp", "localhost:0")
			So(err, ShouldBeNil)
			_, port, err := net.SplitHostPort(listener.Addr().String())
			So(err, ShouldBeNil)

			db := database.GetTestDatabase()
			otherAgent := &model.LocalAgent{
				Name:        "other_agent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":9999}`),
			}
			So(db.Create(otherAgent), ShouldBeNil)

			agent := &model.LocalAgent{
				Name:        "test_sftp_server",
				Protocol:    "sftp",
				Paths:       &model.ServerPaths{Root: root},
				ProtoConfig: []byte(`{"address":"localhost","port":` + port + `}`),
			}
			So(db.Create(agent), ShouldBeNil)
			var protoConfig config.SftpProtoConfig
			So(json.Unmarshal(agent.ProtoConfig, &protoConfig), ShouldBeNil)

			pwd := "tata"
			user := &model.LocalAccount{
				LocalAgentID: agent.ID,
				Login:        "toto",
				Password:     []byte(pwd),
			}
			So(db.Create(user), ShouldBeNil)

			otherUser := &model.LocalAccount{
				LocalAgentID: otherAgent.ID,
				Login:        user.Login,
				Password:     []byte("passwd"),
			}
			So(db.Create(otherUser), ShouldBeNil)

			cert := &model.Cert{
				OwnerType:   agent.TableName(),
				OwnerID:     agent.ID,
				Name:        "test_sftp_server_cert",
				PrivateKey:  testPK,
				PublicKey:   testPBK,
				Certificate: []byte("cert"),
			}
			So(db.Create(cert), ShouldBeNil)

			receive := &model.Rule{
				Name:     "receive",
				Comment:  "",
				IsSend:   false,
				Path:     "/receive",
				InPath:   "rcv_in",
				WorkPath: "rcv_tmp",
			}
			send := &model.Rule{
				Name:     "send",
				Comment:  "",
				IsSend:   true,
				Path:     "/send",
				OutPath:  "snd_out",
				WorkPath: "snd_tmp",
			}
			So(db.Create(receive), ShouldBeNil)
			So(db.Create(send), ShouldBeNil)

			serverConfig, err := getSSHServerConfig(db, cert, &protoConfig, agent)
			So(err, ShouldBeNil)

			ctx, cancel := context.WithCancel(context.Background())

			sshList := &sshListener{
				DB:          db,
				Logger:      logger,
				Agent:       agent,
				ProtoConfig: &protoConfig,
				GWConf:      &conf.ServerConfig{Paths: conf.PathsConfig{GatewayHome: root}},
				SSHConf:     serverConfig,
				Listener:    listener,
				connWg:      sync.WaitGroup{},
				ctx:         ctx,
				cancel:      cancel,
			}
			sshList.listen()
			Reset(func() {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				So(sshList.close(ctx), ShouldBeNil)
			})

			Convey("Given that the server shuts down", func() {

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

					client, err := sftp.NewClient(conn)
					So(err, ShouldBeNil)

					Convey("Given an incoming transfer", func() {
						dst, err := client.Create(path.Join(receive.Path, "/test_in_shutdown.dst"))
						So(err, ShouldBeNil)

						Convey("When the server shuts down", func() {
							_, err := dst.Write([]byte{'a'})
							So(err, ShouldBeNil)

							ctx, cancel := context.WithTimeout(context.Background(), time.Second)
							defer cancel()
							So(sshList.close(ctx), ShouldBeNil)

							_, err = dst.Write([]byte{'b'})
							So(err, ShouldNotBeNil)

							Convey("Then the transfer should appear interrupted", func() {
								var t []model.Transfer
								So(db.Select(&t, nil), ShouldBeNil)
								So(t, ShouldNotBeEmpty)

								trans := model.Transfer{
									ID:         t[0].ID,
									Start:      t[0].Start,
									IsServer:   true,
									AccountID:  user.ID,
									AgentID:    agent.ID,
									SourceFile: "test_in_shutdown.dst",
									DestFile:   "test_in_shutdown.dst",
									RuleID:     receive.ID,
									Status:     model.StatusInterrupted,
									Step:       model.StepData,
									Owner:      database.Owner,
									Progress:   1,
								}
								So(t[0], ShouldResemble, trans)
							})
						})
					})

					Convey("Given an outgoing transfer", func() {
						content := []byte("Test outgoing file content")
						file := filepath.Join(root, send.OutPath, "test_out_shutdown.src")

						So(os.MkdirAll(filepath.Join(root, send.OutPath), 0700), ShouldBeNil)
						So(ioutil.WriteFile(file, content, 0600), ShouldBeNil)

						src, err := client.Open(path.Join(send.Path, "test_out_shutdown.src"))
						So(err, ShouldBeNil)

						Convey("When the server shuts down", func() {
							_, err := src.Read(make([]byte, 1))
							So(err, ShouldBeNil)

							ctx, cancel := context.WithTimeout(context.Background(), time.Second)
							defer cancel()
							So(sshList.close(ctx), ShouldBeNil)

							_, err = src.Read(make([]byte, 1))
							So(err, ShouldNotBeNil)

							Convey("Then the transfer should appear interrupted", func() {
								var t []model.Transfer
								So(db.Select(&t, nil), ShouldBeNil)
								So(t, ShouldNotBeEmpty)

								trans := model.Transfer{
									ID:         t[0].ID,
									Start:      t[0].Start,
									IsServer:   true,
									AccountID:  user.ID,
									AgentID:    agent.ID,
									SourceFile: "test_out_shutdown.src",
									DestFile:   "test_out_shutdown.src",
									RuleID:     send.ID,
									Status:     model.StatusInterrupted,
									Step:       model.StepData,
									Owner:      database.Owner,
									Progress:   1,
								}
								So(t[0], ShouldResemble, trans)
							})
						})
					})
				})
			})

			Convey("Given a working server", func() {

				Reset(func() {
					ctx, cancel := context.WithTimeout(context.Background(), time.Second)
					defer cancel()
					So(sshList.close(ctx), ShouldBeNil)
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

						dir := filepath.Join(root, receive.InPath)
						err := os.MkdirAll(dir, 0700)
						So(err, ShouldBeNil)

						Convey("Given that the transfer finishes normally", func() {
							src := bytes.NewBuffer([]byte(content))
							file := "test_in.dst"

							dst, err := client.Create(path.Join(receive.Path, file))
							So(err, ShouldBeNil)

							_, err = dst.ReadFrom(src)
							So(err, ShouldBeNil)

							So(dst.Close(), ShouldBeNil)
							So(client.Close(), ShouldBeNil)
							So(conn.Close(), ShouldBeNil)

							Convey("Then the destination file should exist", func() {
								dest := filepath.Join(dir, file)
								_, err := os.Stat(dest)
								So(err, ShouldBeNil)

								Convey("Then the file's content should be identical "+
									"to the original", func() {
									dstContent, err := ioutil.ReadFile(dest)
									So(err, ShouldBeNil)

									So(string(dstContent), ShouldEqual, content)
								})
							})

							Convey("Then the transfer should appear in the history", func() {
								hist := &model.TransferHistory{
									IsServer:       true,
									IsSend:         receive.IsSend,
									Account:        user.Login,
									Agent:          agent.Name,
									Protocol:       "sftp",
									SourceFilename: file,
									DestFilename:   file,
									Rule:           receive.Name,
									Status:         model.StatusDone,
								}

								ok, err := db.Exists(hist)
								So(err, ShouldBeNil)
								So(ok, ShouldBeTrue)
							})
						})

						Convey("Given that 2 transfers launch simultaneously", func() {
							src1 := bytes.NewBuffer([]byte(content))
							src2 := bytes.NewBuffer([]byte(content))

							dst1, err := client.Create(path.Join(receive.Path, "test_in_1.dst"))
							So(err, ShouldBeNil)
							dst2, err := client.Create(path.Join(receive.Path, "test_in_2.dst"))
							So(err, ShouldBeNil)

							res1 := make(chan error)
							res2 := make(chan error)
							go func() {
								_, err := dst1.ReadFrom(src1)
								res1 <- err
							}()
							go func() {
								_, err := dst2.ReadFrom(src2)
								res2 <- err
							}()

							So(<-res1, ShouldBeNil)
							So(<-res2, ShouldBeNil)

							So(dst1.Close(), ShouldBeNil)
							So(dst2.Close(), ShouldBeNil)

							Convey("Then the destination files should exist", func() {
								path1 := filepath.Join(root, receive.InPath, "test_in_1.dst")
								_, err := os.Stat(path1)
								So(err, ShouldBeNil)

								path2 := filepath.Join(root, receive.InPath, "test_in_2.dst")
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
									IsSend:         receive.IsSend,
									Account:        user.Login,
									Agent:          agent.Name,
									Protocol:       "sftp",
									SourceFilename: "test_in_1.dst",
									DestFilename:   "test_in_1.dst",
									Rule:           receive.Name,
									Status:         model.StatusDone,
								}
								hist2 := &model.TransferHistory{
									IsServer:       true,
									IsSend:         receive.IsSend,
									Account:        user.Login,
									Agent:          agent.Name,
									Protocol:       "sftp",
									SourceFilename: "test_in_2.dst",
									DestFilename:   "test_in_2.dst",
									Rule:           receive.Name,
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
							src := bytes.NewBufferString("test fail content")

							dst, err := client.Create(path.Join(receive.Path, "test_in_fail.dst"))
							So(err, ShouldBeNil)

							_, err = dst.Write(src.Next(1))
							So(err, ShouldBeNil)
							So(conn.Close(), ShouldBeNil)
							_, err = dst.ReadFrom(src)

							Convey("Then it should return an error", func() {
								So(err, ShouldNotBeNil)

								Convey("Then the transfer should appear in the history", func() {
									time.Sleep(100 * time.Millisecond)

									var t []model.Transfer
									So(db.Select(&t, nil), ShouldBeNil)
									So(t, ShouldBeEmpty)

									var h []model.TransferHistory
									So(db.Select(&h, nil), ShouldBeNil)
									So(h, ShouldNotBeEmpty)

									hist := model.TransferHistory{
										ID:             h[0].ID,
										Owner:          database.Owner,
										IsServer:       true,
										IsSend:         receive.IsSend,
										Account:        user.Login,
										Agent:          agent.Name,
										Protocol:       "sftp",
										SourceFilename: "test_in_fail.dst",
										DestFilename:   "test_in_fail.dst",
										Rule:           receive.Name,
										Start:          h[0].Start,
										Stop:           h[0].Stop,
										Status:         model.StatusError,
										Step:           model.StepData,
										Error: model.NewTransferError(model.TeConnectionReset,
											"SFTP connection closed unexpectedly"),
										Progress: 1,
									}
									So(h[0], ShouldResemble, hist)
								})
							})
						})

						Convey("Given that the account is not authorized to use the rule", func() {
							other := &model.LocalAccount{
								LocalAgentID: agent.ID,
								Login:        "other",
								Password:     []byte("password"),
							}
							So(db.Create(other), ShouldBeNil)

							access := &model.RuleAccess{
								RuleID:     receive.ID,
								ObjectID:   other.ID,
								ObjectType: other.TableName(),
							}
							So(db.Create(access), ShouldBeNil)

							Convey("When starting a transfer", func() {
								_, err := client.Create(path.Join(receive.Path, "test_in.dst"))

								Convey("Then it should return an error", func() {
									So(err, ShouldBeError, `sftp: "Permission Denied"`+
										` (SSH_FX_PERMISSION_DENIED)`)
								})
							})
						})
					})

					Convey("Given an outgoing transfer", func() {
						file := filepath.Join(root, send.OutPath, "test_out.src")
						content := []byte("Test outgoing file")

						So(os.MkdirAll(filepath.Join(root, send.OutPath), 0700), ShouldBeNil)
						So(ioutil.WriteFile(file, content, 0600), ShouldBeNil)

						Convey("Given that the transfer finishes normally", func() {
							src, err := client.Open(path.Join(send.Path, "test_out.src"))
							So(err, ShouldBeNil)

							dst := &bytes.Buffer{}
							n, err := src.WriteTo(dst)
							So(err, ShouldBeNil)
							So(n, ShouldEqual, len(content))

							So(src.Close(), ShouldBeNil)
							So(client.Close(), ShouldBeNil)
							So(conn.Close(), ShouldBeNil)

							Convey("Then the file's content should be identical "+
								"to the original", func() {
								So(dst.String(), ShouldEqual, string(content))
							})

							Convey("Then the transfer should appear in the history", func() {
								hist := &model.TransferHistory{
									IsServer:       true,
									IsSend:         send.IsSend,
									Account:        user.Login,
									Agent:          agent.Name,
									Protocol:       "sftp",
									SourceFilename: "test_out.src",
									DestFilename:   "test_out.src",
									Rule:           send.Name,
									Status:         model.StatusDone,
								}

								ok, err := db.Exists(hist)
								So(err, ShouldBeNil)
								So(ok, ShouldBeTrue)
							})
						})

						Convey("Given that the transfer fails", func() {
							src, err := client.Open(path.Join(send.Path, "test_out.src"))
							So(err, ShouldBeNil)

							dst := ioutil.Discard

							_, err = src.Read(make([]byte, 1))
							So(err, ShouldBeNil)
							So(conn.Close(), ShouldBeNil)
							_, err = src.WriteTo(dst)

							Convey("Then it should return an error", func() {
								So(err, ShouldNotBeNil)

								Convey("Then the transfer should appear in the history", func() {
									time.Sleep(100 * time.Millisecond)

									var t []model.Transfer
									So(db.Select(&t, nil), ShouldBeNil)
									So(t, ShouldBeEmpty)

									var h []model.TransferHistory
									So(db.Select(&h, nil), ShouldBeNil)
									So(h, ShouldNotBeEmpty)

									hist := model.TransferHistory{
										ID:             h[0].ID,
										Owner:          database.Owner,
										IsServer:       true,
										IsSend:         send.IsSend,
										Account:        user.Login,
										Agent:          agent.Name,
										Protocol:       "sftp",
										SourceFilename: "test_out.src",
										DestFilename:   "test_out.src",
										Rule:           send.Name,
										Start:          h[0].Start,
										Stop:           h[0].Stop,
										Status:         model.StatusError,
										Step:           model.StepData,
										Error: model.NewTransferError(model.TeConnectionReset,
											"SFTP connection closed unexpectedly"),
										Progress: 1,
									}
									So(h[0], ShouldResemble, hist)
								})
							})
						})
					})
				})
			})
		})
	})
}
