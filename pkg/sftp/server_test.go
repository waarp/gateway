package sftp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
	"github.com/pkg/sftp"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/ssh"
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
		root := utils.SlashJoin(Dir, "root")

		agent := &model.LocalAgent{
			Name:        "test_sftp_server",
			Protocol:    "sftp",
			Root:        root,
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
	dir := "test_ssh_server"
	root, err := filepath.Abs(dir)
	if err != nil {
		t.FailNow()
	}

	Convey("Given an SFTP server", t, func() {
		So(os.Mkdir(root, 0700), ShouldBeNil)
		Reset(func() { _ = os.RemoveAll(root) })

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
			Root:        root,
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
			Path:     "receive",
			InPath:   "receive",
			WorkPath: "receive/tmp",
		}
		send := &model.Rule{
			Name:     "send",
			Comment:  "",
			IsSend:   true,
			Path:     "send",
			OutPath:  "send",
			WorkPath: "send/tmp",
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
			GWConf:      &conf.ServerConfig{Paths: conf.PathsConfig{GatewayHome: Dir}},
			SSHConf:     serverConfig,
			Listener:    listener,
			connWg:      sync.WaitGroup{},
			ctx:         ctx,
			cancel:      cancel,
		}
		sshList.listen()

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
					dst, err := client.Create(receive.Path + "/test_in_shutdown.dst")
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
								SourceFile: ".",
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
					content := "Test outgoing file "

					err := os.MkdirAll(root+send.Path, 0700)
					So(err, ShouldBeNil)

					err = ioutil.WriteFile(root+send.Path+"/test_out_shutdown.src",
						[]byte(content), 0600)
					So(err, ShouldBeNil)

					Reset(func() { _ = os.RemoveAll(root + send.Path) })

					src, err := client.Open(send.Path + "/test_out_shutdown.src")
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
								DestFile:   ".",
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

					dir := utils.SlashJoin(root, receive.Path)
					err := os.MkdirAll(dir, 0700)
					So(err, ShouldBeNil)

					Reset(func() { _ = os.RemoveAll(dir) })

					Convey("Given that the transfer finishes normally", func() {
						src := bytes.NewBuffer([]byte(content))
						file := "test_in.dst"

						dst, err := client.Create(utils.SlashJoin(receive.Path, file))
						So(err, ShouldBeNil)

						_, err = io.Copy(dst, src)
						So(err, ShouldBeNil)

						So(dst.Close(), ShouldBeNil)
						So(client.Close(), ShouldBeNil)
						So(conn.Close(), ShouldBeNil)

						Convey("Then the destination file should exist", func() {
							path := utils.SlashJoin(dir, file)
							_, err := os.Stat(path)
							So(err, ShouldBeNil)

							Convey("Then the file's content should be identical "+
								"to the original", func() {
								dstContent, err := ioutil.ReadFile(path)
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
								SourceFilename: ".",
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

						dst1, err := client.Create(receive.Path + "/test_in_1.dst")
						So(err, ShouldBeNil)
						dst2, err := client.Create(receive.Path + "/test_in_2.dst")
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
							path1 := root + receive.Path + "/test_in_1.dst"
							_, err := os.Stat(path1)
							So(err, ShouldBeNil)

							path2 := root + receive.Path + "/test_in_2.dst"
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
								SourceFilename: ".",
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
								SourceFilename: ".",
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

						dst, err := client.Create(receive.Path + "/test_in_fail.dst")
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
									SourceFilename: ".",
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
							_, err := client.Create(receive.Path + "/test_in.dst")

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, `sftp: "Permission Denied"`+
									` (SSH_FX_PERMISSION_DENIED)`)
							})
						})
					})
				})

				Convey("Given an outgoing transfer", func() {
					content := "Test outgoing file"

					err := os.MkdirAll(root+send.Path, 0700)
					So(err, ShouldBeNil)

					err = ioutil.WriteFile(root+send.Path+"/test_out.src", []byte(content), 0600)
					So(err, ShouldBeNil)

					Reset(func() { _ = os.RemoveAll(root + send.Path) })

					Convey("Given that the transfer finishes normally", func() {
						src, err := client.Open(send.Path + "/test_out.src")
						So(err, ShouldBeNil)

						dst := &bytes.Buffer{}

						_, err = src.WriteTo(dst)
						So(err, ShouldBeNil)

						So(src.Close(), ShouldBeNil)
						So(client.Close(), ShouldBeNil)
						So(conn.Close(), ShouldBeNil)

						Convey("Then the file's content should be identical "+
							"to the original", func() {
							So(dst.String(), ShouldEqual, content)
						})

						Convey("Then the transfer should appear in the history", func() {
							hist := &model.TransferHistory{
								IsServer:       true,
								IsSend:         send.IsSend,
								Account:        user.Login,
								Agent:          agent.Name,
								Protocol:       "sftp",
								SourceFilename: "test_out.src",
								DestFilename:   ".",
								Rule:           send.Name,
								Status:         model.StatusDone,
							}

							ok, err := db.Exists(hist)
							So(err, ShouldBeNil)
							So(ok, ShouldBeTrue)
						})
					})

					Convey("Given that the transfer fails", func() {
						src, err := client.Open(send.Path + "/test_out.src")
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
									DestFilename:   ".",
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
}

func TestSSHServerTasks(t *testing.T) {
	logger := log.NewLogger("test_sftp_server")

	Convey("Given an SFTP server", t, func() {
		root, err := ioutil.TempDir("", "gateway-test")
		So(err, ShouldBeNil)
		Reset(func() { _ = os.RemoveAll(root) })

		listener, err := net.Listen("tcp", "localhost:0")
		So(err, ShouldBeNil)
		_, port, err := net.SplitHostPort(listener.Addr().String())
		So(err, ShouldBeNil)

		db := database.GetTestDatabase()
		agent := &model.LocalAgent{
			Name:        "test_sftp_server",
			Protocol:    "sftp",
			Root:        root,
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

		cert := &model.Cert{
			OwnerType:   agent.TableName(),
			OwnerID:     agent.ID,
			Name:        "test_sftp_server_cert",
			PrivateKey:  testPK,
			PublicKey:   testPBK,
			Certificate: []byte("cert"),
		}
		So(db.Create(cert), ShouldBeNil)

		serverConfig, err := getSSHServerConfig(db, cert, &protoConfig, agent)
		So(err, ShouldBeNil)

		ctx, cancel := context.WithCancel(context.Background())

		sshList := &sshListener{
			DB:          db,
			Logger:      logger,
			Agent:       agent,
			ProtoConfig: &protoConfig,
			GWConf:      &conf.ServerConfig{Paths: conf.PathsConfig{GatewayHome: Dir}},
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
				receive := &model.Rule{
					Name:   "receive",
					IsSend: false,
					Path:   "/receive",
				}
				So(db.Create(receive), ShouldBeNil)

				content := "Test incoming file"

				err := os.MkdirAll(root+receive.Path, 0700)
				So(err, ShouldBeNil)

				Reset(func() { _ = os.RemoveAll(root + receive.Path) })

				Convey("Given that the preTasks succeed", func() {
					task := &model.Task{
						RuleID: receive.ID,
						Chain:  model.ChainPre,
						Rank:   0,
						Type:   "TESTSUCCESS",
						Args:   []byte("{}"),
					}
					So(db.Create(task), ShouldBeNil)

					Convey("Given that the transfer finishes normally", func() {
						src := bytes.NewBuffer([]byte(content))

						dst, err := client.Create(receive.Path + "/test_in.dst")
						So(err, ShouldBeNil)

						_, err = dst.ReadFrom(src)
						So(err, ShouldBeNil)

						So(dst.Close(), ShouldBeNil)
						So(client.Close(), ShouldBeNil)
						So(conn.Close(), ShouldBeNil)

						Convey("Then the transfer should have succeeded", func() {
							hist := &model.TransferHistory{
								IsServer:       true,
								IsSend:         receive.IsSend,
								Account:        user.Login,
								Agent:          agent.Name,
								Protocol:       "sftp",
								SourceFilename: ".",
								DestFilename:   "test_in.dst",
								Rule:           receive.Name,
								Status:         model.StatusDone,
							}

							ok, err := db.Exists(hist)
							So(err, ShouldBeNil)
							So(ok, ShouldBeTrue)
						})
					})
				})

				Convey("Given that the preTasks fails", func() {
					task := &model.Task{
						RuleID: receive.ID,
						Chain:  model.ChainPre,
						Rank:   0,
						Type:   "TESTFAIL",
						Args:   []byte("{}"),
					}
					So(db.Create(task), ShouldBeNil)

					Convey("Given that the transfer finishes normally", func() {
						_, err := client.Create(receive.Path + "/test_in.dst")
						So(err, ShouldNotBeNil)

						Convey("Then the transfer should have failed", func() {
							hist := &model.TransferHistory{
								IsServer:       true,
								IsSend:         receive.IsSend,
								Account:        user.Login,
								Agent:          agent.Name,
								Protocol:       "sftp",
								SourceFilename: ".",
								DestFilename:   "test_in.dst",
								Rule:           receive.Name,
								Status:         model.StatusError,
							}

							err := client.Close()
							So(err, ShouldBeNil)

							So(db.Get(hist), ShouldBeNil)
							So(hist.Error, ShouldResemble, model.NewTransferError(
								model.TeExternalOperation, "Task TESTFAIL @ receive PRE[0]: task failed"))
						})
					})
				})

				Convey("Given that the postTasks succeed", func() {
					task := &model.Task{
						RuleID: receive.ID,
						Chain:  model.ChainPost,
						Rank:   0,
						Type:   "TESTSUCCESS",
						Args:   []byte("{}"),
					}
					So(db.Create(task), ShouldBeNil)

					Convey("Given that the transfer finishes normally", func() {
						src := bytes.NewBuffer([]byte(content))

						dst, err := client.Create(receive.Path + "/test_in.dst")
						So(err, ShouldBeNil)

						_, err = dst.ReadFrom(src)
						So(err, ShouldBeNil)

						So(dst.Close(), ShouldBeNil)
						So(client.Close(), ShouldBeNil)
						So(conn.Close(), ShouldBeNil)

						Convey("Then the transfer should have succeeded", func() {
							hist := &model.TransferHistory{
								IsServer:       true,
								IsSend:         receive.IsSend,
								Account:        user.Login,
								Agent:          agent.Name,
								Protocol:       "sftp",
								SourceFilename: ".",
								DestFilename:   "test_in.dst",
								Rule:           receive.Name,
								Status:         model.StatusDone,
							}

							ok, err := db.Exists(hist)
							So(err, ShouldBeNil)
							So(ok, ShouldBeTrue)
						})
					})
				})

				Convey("Given that the postTasks fails", func() {
					task := &model.Task{
						RuleID: receive.ID,
						Chain:  model.ChainPost,
						Rank:   0,
						Type:   "TESTFAIL",
						Args:   []byte("{}"),
					}
					So(db.Create(task), ShouldBeNil)

					Convey("Given that the transfer finishes normally", func() {
						src := bytes.NewBuffer([]byte(content))

						dst, err := client.Create(receive.Path + "/test_in.dst")
						So(err, ShouldBeNil)

						_, err = dst.ReadFrom(src)
						So(err, ShouldBeNil)

						err = dst.Close()

						Convey("Then it should return an error", func() {
							So(err, ShouldNotBeNil)
						})

						So(client.Close(), ShouldBeNil)
						So(conn.Close(), ShouldBeNil)

						Convey("Then the transfer should have failed", func() {
							hist := &model.TransferHistory{
								IsServer:       true,
								IsSend:         receive.IsSend,
								Account:        user.Login,
								Agent:          agent.Name,
								Protocol:       "sftp",
								SourceFilename: ".",
								DestFilename:   "test_in.dst",
								Rule:           receive.Name,
								Status:         model.StatusError,
							}

							So(db.Get(hist), ShouldBeNil)
							So(hist.Error, ShouldResemble, model.NewTransferError(
								model.TeExternalOperation, "Task TESTFAIL @ receive POST[0]: task failed"))
						})
					})
				})
			})

			Convey("Given an outgoing transfer", func() {
				send := &model.Rule{
					Name:    "send",
					IsSend:  true,
					Path:    "/send",
					OutPath: "/send",
				}
				So(db.Create(send), ShouldBeNil)

				content := "Test outgoing file"

				path := utils.SlashJoin(root, send.Path)
				So(os.MkdirAll(path, 0700), ShouldBeNil)

				srcFile := "test_out.src"
				srcFilepath := utils.SlashJoin(path, srcFile)
				So(ioutil.WriteFile(srcFilepath, []byte(content), 0600), ShouldBeNil)

				Reset(func() { _ = os.RemoveAll(root + send.Path) })

				Convey("Given that the preTasks succeed", func() {
					task := &model.Task{
						RuleID: send.ID,
						Chain:  model.ChainPre,
						Rank:   0,
						Type:   "TESTSUCCESS",
						Args:   []byte("{}"),
					}
					So(db.Create(task), ShouldBeNil)

					Convey("Given that the transfer finishes normally", func() {
						reqPath := utils.SlashJoin(send.Path, srcFile)
						src, err := client.Open(reqPath)
						So(err, ShouldBeNil)

						dst := &bytes.Buffer{}

						_, err = src.WriteTo(dst)
						So(err, ShouldBeNil)

						So(src.Close(), ShouldBeNil)
						So(client.Close(), ShouldBeNil)
						So(conn.Close(), ShouldBeNil)

						Convey("Then the dest file should contain the source's content", func() {
							So(dst.String(), ShouldEqual, content)
						})

						Convey("Then the transfer should have succeeded", func() {
							hist := &model.TransferHistory{
								IsServer:       true,
								IsSend:         send.IsSend,
								Account:        user.Login,
								Agent:          agent.Name,
								Protocol:       "sftp",
								SourceFilename: srcFile,
								DestFilename:   ".",
								Rule:           send.Name,
								Status:         model.StatusDone,
							}

							ok, err := db.Exists(hist)
							So(err, ShouldBeNil)
							So(ok, ShouldBeTrue)
						})
					})
				})

				Convey("Given that the preTasks fails", func() {
					task := &model.Task{
						RuleID: send.ID,
						Chain:  model.ChainPre,
						Rank:   0,
						Type:   "TESTFAIL",
						Args:   []byte("{}"),
					}
					So(db.Create(task), ShouldBeNil)

					Convey("Given that the transfer finishes normally", func() {
						_, err := client.Open(send.Path + "/test_out.src")
						So(err, ShouldNotBeNil)

						So(client.Close(), ShouldBeNil)
						So(conn.Close(), ShouldBeNil)

						Convey("Then the transfer should have failed", func() {
							hist := &model.TransferHistory{
								IsServer:       true,
								IsSend:         send.IsSend,
								Account:        user.Login,
								Agent:          agent.Name,
								Protocol:       "sftp",
								SourceFilename: "test_out.src",
								DestFilename:   ".",
								Rule:           send.Name,
								Status:         model.StatusError,
							}

							So(db.Get(hist), ShouldBeNil)
							So(hist.Error, ShouldResemble, model.NewTransferError(
								model.TeExternalOperation, "Task TESTFAIL @ send"+
									" PRE[0]: task failed"))
						})
					})
				})

				Convey("Given that the postTasks succeed", func() {
					task := &model.Task{
						RuleID: send.ID,
						Chain:  model.ChainPost,
						Rank:   0,
						Type:   "TESTSUCCESS",
						Args:   []byte("{}"),
					}
					So(db.Create(task), ShouldBeNil)

					Convey("Given that the transfer finishes normally", func() {
						src, err := client.Open(send.Path + "/test_out.src")
						So(err, ShouldBeNil)

						dst := &bytes.Buffer{}

						_, err = src.WriteTo(dst)
						So(err, ShouldBeNil)

						So(src.Close(), ShouldBeNil)
						So(client.Close(), ShouldBeNil)
						So(conn.Close(), ShouldBeNil)

						Convey("Then the dest file should contain the source's content", func() {
							So(dst.String(), ShouldEqual, content)
						})

						Convey("Then the transfer should have succeeded", func() {
							hist := &model.TransferHistory{
								IsServer:       true,
								IsSend:         send.IsSend,
								Account:        user.Login,
								Agent:          agent.Name,
								Protocol:       "sftp",
								SourceFilename: "test_out.src",
								DestFilename:   ".",
								Rule:           send.Name,
								Status:         model.StatusDone,
							}

							ok, err := db.Exists(hist)
							So(err, ShouldBeNil)
							So(ok, ShouldBeTrue)
						})
					})
				})

				Convey("Given that the postTasks fails", func() {
					task := &model.Task{
						RuleID: send.ID,
						Chain:  model.ChainPost,
						Rank:   0,
						Type:   "TESTFAIL",
						Args:   []byte("{}"),
					}
					So(db.Create(task), ShouldBeNil)

					Convey("Given that the transfer finishes normally", func() {
						src, err := client.Open(send.Path + "/test_out.src")
						So(err, ShouldBeNil)

						dst := &bytes.Buffer{}

						_, err = src.WriteTo(dst)
						So(err, ShouldBeNil)

						err = src.Close()

						Convey("Then it should return an error", func() {
							So(err, ShouldNotBeNil)
						})

						So(client.Close(), ShouldBeNil)
						So(conn.Close(), ShouldBeNil)

						Convey("Then the dest file should contain the source's content", func() {
							So(dst.String(), ShouldEqual, content)
						})

						Convey("Then the transfer should have failed", func() {
							hist := &model.TransferHistory{
								IsServer:       true,
								IsSend:         send.IsSend,
								Account:        user.Login,
								Agent:          agent.Name,
								Protocol:       "sftp",
								SourceFilename: "test_out.src",
								DestFilename:   ".",
								Rule:           send.Name,
								Status:         model.StatusError,
							}

							So(db.Get(hist), ShouldBeNil)
							So(hist.Error, ShouldResemble, model.NewTransferError(
								model.TeExternalOperation, "Task TESTFAIL @ send"+
									" POST[0]: task failed"))
						})
					})
				})
			})
		})
	})
}
