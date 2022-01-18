package sftp

import (
	"context"
	"fmt"
	"net"
	"path"
	"testing"
	"time"

	"github.com/pkg/sftp"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/ssh"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/fstest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

//nolint:maintidx //FIXME factorize the function if possible to improve maintainability
func TestSFTPList(t *testing.T) {
	Convey("Given a SFTP server", t, func(c C) {
		testFS := fstest.InitMemFS(c)
		root := "memory:/test_list_root"
		rootPath := mkURL(root)
		db := database.TestDatabase(c)
		conf.GlobalConfig.Paths.GatewayHome = root

		Convey("Given an SFTP server", func(c C) {
			port := getTestPort()
			listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
			So(err, ShouldBeNil)

			addr := listener.Addr().String()

			agent := &model.LocalAgent{
				Name: "test_sftp_server", Protocol: SFTP,
				RootDir: root, Address: types.Addr("localhost", port),
			}
			So(db.Insert(agent).Run(), ShouldBeNil)

			toto := &model.LocalAccount{
				LocalAgentID: agent.ID,
				Login:        "toto",
			}
			So(db.Insert(toto).Run(), ShouldBeNil)

			totoPswd := &model.Credential{
				LocalAccountID: utils.NewNullInt64(toto.ID),
				Type:           auth.PasswordHash,
				Value:          "toto",
			}
			So(db.Insert(totoPswd).Run(), ShouldBeNil)

			tata := &model.LocalAccount{
				LocalAgentID: agent.ID,
				Login:        "tata",
			}
			So(db.Insert(tata).Run(), ShouldBeNil)

			tataPswd := &model.Credential{
				LocalAccountID: utils.NewNullInt64(tata.ID),
				Type:           auth.PasswordHash,
				Value:          "tata",
			}
			So(db.Insert(tataPswd).Run(), ShouldBeNil)

			hostKey := &model.Credential{
				LocalAgentID: utils.NewNullInt64(agent.ID),
				Type:         AuthSSHPrivateKey,
				Value:        RSAPk,
			}
			So(db.Insert(hostKey).Run(), ShouldBeNil)

			hostKey.Value = RSAPk
			serverConfig, err := getSSHServerConfig(db, testhelpers.TestLogger(c, "sftp_test"),
				[]*model.Credential{hostKey}, &serverConfig{}, agent)
			So(err, ShouldBeNil)

			logger := testhelpers.TestLogger(c, "test_sftp_list")

			sshList := &sshListener{
				DB:       db,
				Logger:   logger,
				Server:   agent,
				SSHConf:  serverConfig,
				Listener: listener,
			}
			sshList.handlerMaker = sshList.makeHandlers

			go sshList.listen()
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
					Path:    "path1/send1",
				}
				send2 := &model.Rule{
					Name:    "send2",
					Comment: "",
					IsSend:  true,
					Path:    "path2/send2",
				}
				send3 := &model.Rule{
					Name:    "send3",
					Comment: "",
					IsSend:  true,
					Path:    "path3",
				}
				send4 := &model.Rule{
					Name:    "send4",
					Comment: "",
					IsSend:  true,
					Path:    "path1/subdir/send4",
				}
				recv1 := &model.Rule{
					Name:    "recv1",
					Comment: "",
					IsSend:  false,
					Path:    "path3",
				}
				recv2 := &model.Rule{
					Name:    "recv2",
					Comment: "",
					IsSend:  false,
					Path:    "path4",
				}

				So(db.Insert(send1).Run(), ShouldBeNil)
				So(db.Insert(send2).Run(), ShouldBeNil)
				So(db.Insert(send3).Run(), ShouldBeNil)
				So(db.Insert(send4).Run(), ShouldBeNil)
				So(db.Insert(recv1).Run(), ShouldBeNil)
				So(db.Insert(recv2).Run(), ShouldBeNil)

				totoAccess := &model.RuleAccess{
					RuleID:         send1.ID,
					LocalAccountID: utils.NewNullInt64(toto.ID),
				}
				tataAccess := &model.RuleAccess{
					RuleID:         send2.ID,
					LocalAccountID: utils.NewNullInt64(tata.ID),
				}
				serverAccess := &model.RuleAccess{
					RuleID:       send3.ID,
					LocalAgentID: utils.NewNullInt64(agent.ID),
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

					sshClient, err := ssh.Dial("tcp", addr, sshConf)
					So(err, ShouldBeNil)

					defer sshClient.Close()

					client, err := sftp.NewClient(sshClient)
					So(err, ShouldBeNil)

					defer client.Close()

					Convey("When sending a List request at top level", func() {
						list, err := client.ReadDir("/")
						So(err, ShouldBeNil)

						Convey("Then it should return a list of all the authorized rule paths", func() {
							So(len(list), ShouldEqual, 3)
							So(list[0].Name(), ShouldEqual, "path1")
							So(list[1].Name(), ShouldEqual, "path3")
							So(list[2].Name(), ShouldEqual, "path4")
						})
					})

					Convey("When sending a List request on & directory", func() {
						list, err := client.ReadDir("/path1")
						So(err, ShouldBeNil)

						Convey("Then it should return a list of all the authorized rule paths", func() {
							So(len(list), ShouldEqual, 2)
							So(list[0].Name(), ShouldEqual, "send1")
							So(list[1].Name(), ShouldEqual, "subdir")
						})
					})

					Convey("When sending a List with a rule's path", func() {
						So(fs.MkdirAll(testFS, rootPath.JoinPath("out")), ShouldBeNil)
						So(fs.WriteFullFile(testFS,
							rootPath.JoinPath("out", "list_file1"),
							[]byte("Hello world")), ShouldBeNil)
						So(fs.WriteFullFile(testFS,
							rootPath.JoinPath("out", "list_file2"),
							[]byte("Hello world")), ShouldBeNil)

						list, err := client.ReadDir(send1.Path)
						So(err, ShouldBeNil)

						Convey("Then it should return a list of the files in the rule's out dir", func() {
							So(len(list), ShouldEqual, 2)
							So(list[0].Name(), ShouldEqual, "list_file1")
							So(list[1].Name(), ShouldEqual, "list_file2")
						})
					})

					Convey("When sending a List with an unknown path", func() {
						_, err := client.ReadDir("unknown")

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError, "file does not exist")
						})
					})

					Convey("When sending a Stat to an existing file", func() {
						So(fs.MkdirAll(testFS, rootPath.JoinPath("out", "sub")), ShouldBeNil)
						So(fs.WriteFullFile(testFS,
							rootPath.JoinPath("out", "sub", "stat_file"),
							[]byte("Hello world")), ShouldBeNil)

						info, err := client.Stat(path.Join(send1.Path, "sub", "stat_file"))
						So(err, ShouldBeNil)

						Convey("Then it should returns the file's info", func() {
							exp, err := fs.Stat(testFS, rootPath.JoinPath("out", "sub", "stat_file"))
							So(err, ShouldBeNil)
							So(info.Name(), ShouldEqual, exp.Name())
							So(info.Size(), ShouldEqual, exp.Size())
							So(info.Mode(), ShouldEqual, exp.Mode())
						})
					})

					Convey("When sending a Stat to a virtual directory", func() {
						virtDir := path.Dir(send1.Path)
						info, err := client.Stat(virtDir)
						So(err, ShouldBeNil)

						Convey("Then it should returns the directory's info", func() {
							So(info.Name(), ShouldEqual, path.Base(virtDir))
							So(info.Size(), ShouldEqual, 0)
							So(info.Mode(), ShouldEqual, 0o777|fs.ModeDir)
						})
					})

					Convey("When sending a Stat to a real sub-directory", func() {
						So(fs.MkdirAll(testFS, rootPath.JoinPath("out", "sub")), ShouldBeNil)

						info, err := client.Stat(path.Join(send1.Path, "sub"))
						So(err, ShouldBeNil)

						Convey("Then it should returns the directory's info", func() {
							exp, err := fs.Stat(testFS, rootPath.JoinPath("out", "sub"))
							So(err, ShouldBeNil)
							So(info.Name(), ShouldEqual, exp.Name())
							So(info.Size(), ShouldEqual, exp.Size())
							So(info.Mode(), ShouldEqual, exp.Mode())
						})
					})

					Convey("When sending a Stat to an unknown virtual directory", func() {
						_, err := client.Stat("unknown")

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError, "file does not exist")
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
