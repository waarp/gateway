package sftp

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/sftp/internal"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"
	"github.com/pkg/sftp"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/ssh"
)

func TestSFTPList(t *testing.T) {
	logger := log.NewLogger("test_sftp_list_server")

	Convey("Given a SFTP server", t, func(c C) {
		root := testhelpers.TempDir(c, "test_list_root")
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

			toto := &model.LocalAccount{
				LocalAgentID: agent.ID,
				Login:        "toto",
				PasswordHash: hash("toto"),
			}
			So(db.Insert(toto).Run(), ShouldBeNil)

			tata := &model.LocalAccount{
				LocalAgentID: agent.ID,
				Login:        "tata",
				PasswordHash: hash("tata"),
			}
			So(db.Insert(tata).Run(), ShouldBeNil)

			hostKey := model.Crypto{
				OwnerType:  agent.TableName(),
				OwnerID:    agent.ID,
				Name:       "test_sftp_server_key",
				PrivateKey: rsaPK,
			}
			So(db.Insert(&hostKey).Run(), ShouldBeNil)

			serverConfig, err := internal.GetSSHServerConfig(db, []model.Crypto{hostKey}, &protoConfig, agent)
			So(err, ShouldBeNil)

			sshList := &sshListener{
				DB:               db,
				Logger:           logger,
				Agent:            agent,
				ProtoConfig:      &protoConfig,
				SSHConf:          serverConfig,
				Listener:         listener,
				runningTransfers: pipeline.NewTransferMap(),
			}
			sshList.listen()
			Reset(func() {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				So(sshList.close(ctx), ShouldBeNil)
			})

			Convey("Given a few rules with various permissions", func() {
				send1 := &model.Rule{
					Name:    "send1",
					Comment: "",
					IsSend:  true,
					Path:    "/path1",
				}
				send2 := &model.Rule{
					Name:    "send2",
					Comment: "",
					IsSend:  true,
					Path:    "/path2",
				}
				send3 := &model.Rule{
					Name:    "send3",
					Comment: "",
					IsSend:  true,
					Path:    "/path3",
				}
				send4 := &model.Rule{
					Name:    "send4",
					Comment: "",
					IsSend:  true,
					Path:    "/path4",
				}
				recv1 := &model.Rule{
					Name:    "recv1",
					Comment: "",
					IsSend:  false,
					Path:    "/path3",
				}
				recv2 := &model.Rule{
					Name:    "recv2",
					Comment: "",
					IsSend:  false,
					Path:    "/path5",
				}

				So(db.Insert(send1).Run(), ShouldBeNil)
				So(db.Insert(send2).Run(), ShouldBeNil)
				So(db.Insert(send3).Run(), ShouldBeNil)
				So(db.Insert(send4).Run(), ShouldBeNil)
				So(db.Insert(recv1).Run(), ShouldBeNil)
				So(db.Insert(recv2).Run(), ShouldBeNil)

				totoAccess := &model.RuleAccess{
					RuleID:     send1.ID,
					ObjectID:   toto.ID,
					ObjectType: toto.TableName(),
				}
				tataAccess := &model.RuleAccess{
					RuleID:     send2.ID,
					ObjectID:   tata.ID,
					ObjectType: toto.TableName(),
				}
				serverAccess := &model.RuleAccess{
					RuleID:     send3.ID,
					ObjectID:   agent.ID,
					ObjectType: agent.TableName(),
				}
				So(db.Insert(totoAccess).Run(), ShouldBeNil)
				So(db.Insert(tataAccess).Run(), ShouldBeNil)
				So(db.Insert(serverAccess).Run(), ShouldBeNil)

				Convey("Given a SFTP client", func() {
					sshConf := &ssh.ClientConfig{
						User: toto.Login,
						Auth: []ssh.AuthMethod{
							ssh.Password("toto"),
						},
						HostKeyCallback: ssh.InsecureIgnoreHostKey(),
					}
					addr := "localhost:" + port

					conn, err := net.Dial("tcp", addr)
					So(err, ShouldBeNil)
					sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, sshConf)
					So(err, ShouldBeNil)

					sshClient := ssh.NewClient(sshConn, chans, reqs)
					defer sshClient.Close()
					client, err := sftp.NewClient(sshClient)
					So(err, ShouldBeNil)
					defer client.Close()

					Convey("When sending a List request at top level", func() {
						list, err := client.ReadDir("/")
						So(err, ShouldBeNil)

						Convey("Then it should return a list of all the authorized rule paths", func() {
							So(len(list), ShouldEqual, 4)
							So(list[0].Name(), ShouldEqual, "path1")
							So(list[1].Name(), ShouldEqual, "path3")
							So(list[2].Name(), ShouldEqual, "path4")
							So(list[3].Name(), ShouldEqual, "path5")
						})
					})

					Convey("When sending a List with a rule's path", func() {
						So(os.Mkdir(filepath.Join(root, "out"), 0o700), ShouldBeNil)
						So(ioutil.WriteFile(filepath.Join(root, "out", "list_file1"),
							[]byte("Hello world"), 0o600), ShouldBeNil)
						So(ioutil.WriteFile(filepath.Join(root, "out", "list_file2"),
							[]byte("Hello world"), 0o600), ShouldBeNil)

						list, err := client.ReadDir(send1.Path)
						So(err, ShouldBeNil)

						Convey("Then it should return a list of the files in the rule's out dir", func() {
							So(len(list), ShouldEqual, 2)
							So(list[0].Name(), ShouldEqual, "list_file1")
							So(list[1].Name(), ShouldEqual, "list_file2")
						})
					})

					Convey("When sending a List with an unknown path", func() {
						_, err := client.ReadDir("/unknown")

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError, "file does not exist")
						})
					})

					Convey("When sending a Stat to an existing file", func() {
						So(os.Mkdir(filepath.Join(root, "out"), 0o700), ShouldBeNil)
						So(ioutil.WriteFile(filepath.Join(root, "out", "stat_file"),
							[]byte("Hello world"), 0o600), ShouldBeNil)

						info, err := client.Stat(path.Join(send1.Path, "stat_file"))
						So(err, ShouldBeNil)

						Convey("Then it should returns the file's info", func() {
							exp, err := os.Stat(filepath.Join(root, "out", "stat_file"))
							So(err, ShouldBeNil)
							So(info.Name(), ShouldResemble, exp.Name())
							So(info.Size(), ShouldResemble, exp.Size())
							So(info.Mode(), ShouldResemble, exp.Mode())
						})
					})

					Convey("When sending a Stat to an unknown file", func() {
						_, err := client.Stat(path.Join(send1.Path, "unknown"))

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError, "file does not exist")
						})
					})
				})
			})
		})
	})
}
