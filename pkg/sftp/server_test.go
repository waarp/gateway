//+build todo

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

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/sftp/internal"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
	"github.com/pkg/sftp"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/ssh"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"
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
	Convey("Given a running SFTP server service", t, func(dbc C) {
		db := database.TestDatabase(dbc, "ERROR")
		port := getTestPort()

		agent := &model.LocalAgent{
			Name:        "test_sftp_server",
			Protocol:    "sftp",
			ProtoConfig: json.RawMessage(`{}`),
			Address:     "localhost:" + port,
		}
		So(db.Insert(agent).Run(), ShouldBeNil)

		cert := &model.Cert{
			OwnerType:   agent.TableName(),
			OwnerID:     agent.ID,
			Name:        "test_sftp_server_cert",
			PrivateKey:  testPK,
			PublicKey:   testPBK,
			Certificate: []byte("cert"),
		}
		So(db.Insert(cert).Run(), ShouldBeNil)

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
	Convey("Given an SFTP server service", t, func(dbc C) {
		db := database.TestDatabase(dbc, "ERROR")
		port := getTestPort()
		root, err := filepath.Abs("server_start_root")
		So(err, ShouldBeNil)

		agent := &model.LocalAgent{
			Name:        "test_sftp_server",
			Protocol:    "sftp",
			Root:        root,
			ProtoConfig: json.RawMessage(`{}`),
			Address:     "localhost:" + port,
		}
		So(db.Insert(agent).Run(), ShouldBeNil)

		cert := &model.Cert{
			OwnerType:   agent.TableName(),
			OwnerID:     agent.ID,
			Name:        "test_sftp_server_cert",
			PrivateKey:  testPK,
			PublicKey:   testPBK,
			Certificate: []byte("cert"),
		}
		So(db.Insert(cert).Run(), ShouldBeNil)

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

	Convey("Given a server root", t, func(c C) {
		root := testhelpers.TempDir(c, "test_server_root")
		db := database.TestDatabase(c, "ERROR")

		Convey("Given an SFTP server", func() {
			listener, err := net.Listen("tcp", "localhost:0")
			So(err, ShouldBeNil)
			_, port, err := net.SplitHostPort(listener.Addr().String())
			So(err, ShouldBeNil)

			agent := &model.LocalAgent{
				Name:        "test_sftp_server",
				Protocol:    "sftp",
				Root:        root,
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:" + port,
			}
			So(db.Insert(agent).Run(), ShouldBeNil)
			var protoConfig config.SftpProtoConfig
			So(json.Unmarshal(agent.ProtoConfig, &protoConfig), ShouldBeNil)

			pwd := "tata"
			user := &model.LocalAccount{
				LocalAgentID: agent.ID,
				Login:        "toto",
				Password:     []byte(pwd),
			}
			So(db.Insert(user).Run(), ShouldBeNil)

			cert := model.Cert{
				OwnerType:   agent.TableName(),
				OwnerID:     agent.ID,
				Name:        "test_sftp_server_cert",
				PrivateKey:  testPK,
				PublicKey:   testPBK,
				Certificate: []byte("cert"),
			}
			So(db.Insert(&cert).Run(), ShouldBeNil)

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
			So(db.Insert(receive).Run(), ShouldBeNil)
			So(db.Insert(send).Run(), ShouldBeNil)

			serverConfig, err := internal.GetSSHServerConfig(db, []model.Cert{cert}, &protoConfig, agent)
			So(err, ShouldBeNil)

			ctx, cancel := context.WithCancel(context.Background())

			sshList := &SSHListener{
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
				_ = sshList.close(ctx)
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

							sshList.cancel()

							_, err = dst.Write([]byte{'b'})
							So(err, ShouldNotBeNil)

							sshList.connWg.Wait()

							Convey("Then the transfer should appear interrupted", func() {
								var transfers model.Transfers
								So(db.Select(&transfers).Run(), ShouldBeNil)
								So(transfers, ShouldNotBeEmpty)

								trans := model.Transfer{
									ID:        transfers[0].ID,
									Start:     transfers[0].Start,
									IsServer:  true,
									AccountID: user.ID,
									AgentID:   agent.ID,
									TrueFilepath: utils.ToStandardPath(
										filepath.Join(root, receive.WorkPath,
											"test_in_shutdown.dst.tmp")),
									SourceFile: "test_in_shutdown.dst",
									DestFile:   "test_in_shutdown.dst",
									RuleID:     receive.ID,
									Status:     types.StatusInterrupted,
									Step:       types.StepData,
									Owner:      database.Owner,
									Progress:   1,
								}
								So(transfers[0], ShouldResemble, trans)
							})
						})
					})

					Convey("Given an outgoing transfer", func() {
						content := []byte("Test outgoing file content")
						file := filepath.Join(root, send.OutPath, "test_out_shutdown.src")

						So(os.MkdirAll(filepath.Join(root, send.OutPath), 0o700), ShouldBeNil)
						So(ioutil.WriteFile(file, content, 0o600), ShouldBeNil)

						src, err := client.Open(path.Join(send.Path, "test_out_shutdown.src"))
						So(err, ShouldBeNil)

						Convey("When the server shuts down", func() {
							_, err := src.Read(make([]byte, 1))
							So(err, ShouldBeNil)

							sshList.cancel()

							_, err = src.Read(make([]byte, 1))
							So(err, ShouldNotBeNil)

							sshList.connWg.Wait()

							Convey("Then the transfer should appear interrupted", func() {
								var transfers model.Transfers
								So(db.Select(&transfers).Run(), ShouldBeNil)
								So(transfers, ShouldNotBeEmpty)

								trans := model.Transfer{
									ID:        transfers[0].ID,
									Start:     transfers[0].Start,
									IsServer:  true,
									AccountID: user.ID,
									AgentID:   agent.ID,
									TrueFilepath: utils.ToStandardPath(
										filepath.Join(root, send.OutPath,
											"test_out_shutdown.src")),
									SourceFile: "test_out_shutdown.src",
									DestFile:   "test_out_shutdown.src",
									RuleID:     send.ID,
									Status:     types.StatusInterrupted,
									Step:       types.StepData,
									Owner:      database.Owner,
									Progress:   1,
								}
								So(transfers[0], ShouldResemble, trans)
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
						err := os.MkdirAll(dir, 0o700)
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

							Convey("Then the transfer should appear in the history", func(c C) {
								hist := &model.HistoryEntry{}
								So(db.Get(hist, "is_server=? AND is_send=? AND "+
									"account=? AND agent=? AND protocol=? AND "+
									"source_filename=? AND dest_filename=? AND "+
									"rule=? AND status=?", true, receive.IsSend,
									user.Login, agent.Name, "sftp", file, file,
									receive.Name, types.StatusDone).Run(), ShouldBeNil)
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
								So(client.Close(), ShouldBeNil)

								hist1 := &model.HistoryEntry{}
								So(db.Get(hist1, "is_server=? AND is_send=? AND "+
									"account=? AND agent=? AND protocol=? AND "+
									"source_filename=? AND dest_filename=? AND "+
									"rule=? AND status=?", true, receive.IsSend,
									user.Login, agent.Name, "sftp", "test_in_1.dst",
									"test_in_1.dst", receive.Name, types.StatusDone).
									Run(), ShouldBeNil)

								hist2 := &model.HistoryEntry{}
								So(db.Get(hist2, "is_server=? AND is_send=? AND "+
									"account=? AND agent=? AND protocol=? AND "+
									"source_filename=? AND dest_filename=? AND "+
									"rule=? AND status=?", true, receive.IsSend,
									user.Login, agent.Name, "sftp", "test_in_1.dst",
									"test_in_1.dst", receive.Name, types.StatusDone).
									Run(), ShouldBeNil)
							})
						})

						Convey("Given that the transfer fails", func() {
							src := bytes.NewBufferString("test fail content")

							dst, err := client.Create(path.Join(receive.Path, "test_in_fail.dst"))
							So(err, ShouldBeNil)

							_, err = dst.Write(src.Next(1))
							So(err, ShouldBeNil)
							So(conn.Close(), ShouldBeNil)
							_, err = dst.Write(src.Next(1))

							sshList.connWg.Wait()

							Convey("Then it should return an error", func() {
								So(err, ShouldNotBeNil)

								Convey("Then the transfer should appear in the history", func() {
									var transfers model.Transfers
									So(db.Select(&transfers).Run(), ShouldBeNil)
									So(transfers, ShouldHaveLength, 1)

									trans := model.Transfer{
										ID:               transfers[0].ID,
										RemoteTransferID: "",
										Owner:            database.Owner,
										IsServer:         true,
										AccountID:        user.ID,
										AgentID:          agent.ID,
										TrueFilepath: utils.ToStandardPath(filepath.Join(
											root, receive.WorkPath, "test_in_fail.dst.tmp")),
										SourceFile: "test_in_fail.dst",
										DestFile:   "test_in_fail.dst",
										RuleID:     receive.ID,
										Start:      transfers[0].Start,
										Status:     types.StatusError,
										Step:       types.StepData,
										Error: types.NewTransferError(types.TeConnectionReset,
											"SFTP connection closed unexpectedly"),
										Progress: 1,
									}
									So(transfers[0], ShouldResemble, trans)
								})
							})
						})

						Convey("Given that the account is not authorized to use the rule", func() {
							other := &model.LocalAccount{
								LocalAgentID: agent.ID,
								Login:        "other",
								Password:     []byte("password"),
							}
							So(db.Insert(other).Run(), ShouldBeNil)

							access := &model.RuleAccess{
								RuleID:     receive.ID,
								ObjectID:   other.ID,
								ObjectType: other.TableName(),
							}
							So(db.Insert(access).Run(), ShouldBeNil)

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

						So(os.MkdirAll(filepath.Join(root, send.OutPath), 0o700), ShouldBeNil)
						So(ioutil.WriteFile(file, content, 0o600), ShouldBeNil)

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
								hist := &model.HistoryEntry{}
								So(db.Get(hist, "is_server=? AND is_send=? AND "+
									"account=? AND agent=? AND protocol=? AND "+
									"source_filename=? AND dest_filename=? AND "+
									"rule=? AND status=?", true, send.IsSend,
									user.Login, agent.Name, "sftp", "test_out.src",
									"test_out.src", send.Name, types.StatusDone).
									Run(), ShouldBeNil)
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

							sshList.connWg.Wait()

							Convey("Then it should return an error", func() {
								So(err, ShouldNotBeNil)

								Convey("Then the transfer should appear in the history", func() {
									var transfers model.Transfers
									So(db.Select(&transfers).Run(), ShouldBeNil)
									So(transfers, ShouldHaveLength, 1)

									trans := model.Transfer{
										ID:               transfers[0].ID,
										RemoteTransferID: "",
										Owner:            database.Owner,
										IsServer:         true,
										AccountID:        user.ID,
										AgentID:          agent.ID,
										TrueFilepath: utils.ToStandardPath(filepath.Join(
											root, send.OutPath, "test_out.src")),
										SourceFile: "test_out.src",
										DestFile:   "test_out.src",
										RuleID:     send.ID,
										Start:      transfers[0].Start,
										Status:     types.StatusError,
										Step:       types.StepData,
										Error: types.NewTransferError(types.TeConnectionReset,
											"SFTP connection closed unexpectedly"),
										Progress: 1,
									}
									So(transfers[0], ShouldResemble, trans)
								})
							})
						})
					})
				})
			})
		})
	})
}
