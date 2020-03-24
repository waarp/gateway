package pipeline

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
	. "github.com/smartystreets/goconvey/convey"
)

func TestPathIn(t *testing.T) {
	logger := log.NewLogger("test_path_in")

	gwHome := "in_path_test_root"

	cd, err := os.Getwd()
	if err != nil {
		t.FailNow()
	}

	gwRoot := utils.SlashJoin(cd, gwHome)

	Convey("Given a Gateway configuration", t, func() {
		paths := Paths{
			PathsConfig: conf.PathsConfig{GatewayHome: gwRoot},
		}
		Reset(func() { _ = os.RemoveAll(gwRoot) })

		Convey("Given some transfer agents", func() {
			db := database.GetTestDatabase()

			localAgent := &model.LocalAgent{
				Name:        "local_agent",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(localAgent), ShouldBeNil)

			localAccount := &model.LocalAccount{
				LocalAgentID: localAgent.ID,
				Login:        "local_account",
				Password:     []byte("password"),
			}
			So(db.Create(localAccount), ShouldBeNil)

			remoteAgent := &model.RemoteAgent{
				Name:        "remote_agent",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(remoteAgent), ShouldBeNil)

			remoteAccount := &model.RemoteAccount{
				RemoteAgentID: remoteAgent.ID,
				Login:         "remote_account",
				Password:      []byte("password"),
			}
			So(db.Create(remoteAccount), ShouldBeNil)

			testFunc := func(ruleID uint64, workPath, destPath string) {
				Convey("When creating & starting a transfer stream", func() {
					trans := model.Transfer{
						RuleID:     ruleID,
						IsServer:   true,
						AgentID:    localAgent.ID,
						AccountID:  localAccount.ID,
						SourceFile: "file.src",
						DestFile:   "file.dst",
					}

					stream, err := NewTransferStream(context.Background(),
						logger, db, paths, trans)
					So(err, ShouldBeNil)

					So(stream.Start(), ShouldBeNil)

					Convey("Then it should have created the correct work file", func() {
						_, err := os.Stat(utils.SlashJoin(workPath, trans.DestFile))
						So(err, ShouldBeNil)
					})

					Convey("When finalizing the transfer", func() {
						So(stream.Finalize(), ShouldBeNil)

						Convey("Then it should have moved the file to its destination", func() {
							_, err := os.Stat(utils.SlashJoin(destPath, trans.DestFile))
							So(err, ShouldBeNil)
						})
					})
				})
			}

			Convey("Given that it has both a 'in' and 'work' directory", func() {
				inDir := "in"
				workDir := "tmp"
				paths.InDirectory = utils.SlashJoin(gwRoot, inDir)
				paths.WorkDirectory = utils.SlashJoin(gwRoot, workDir)

				Convey("Given a server with a root & work directory", func() {
					serverDir := "server_root"
					serverWork := serverDir + "/server_work"
					paths.ServerRoot = utils.SlashJoin(gwRoot, serverDir)
					paths.ServerWork = utils.SlashJoin(gwRoot, serverWork)

					Convey("Given that the rule has both an 'in' and 'work' directory", func() {
						receive := &model.Rule{
							Name:     "receive",
							IsSend:   false,
							Path:     "path",
							InPath:   "rule_in",
							WorkPath: "rule_work",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverDir, receive.WorkPath)
						destPath := utils.SlashJoin(cd, gwHome, serverDir, receive.InPath)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule only has an 'in' directory", func() {
						receive := &model.Rule{
							Name:   "receive",
							IsSend: false,
							Path:   "path",
							InPath: "rule_in",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverWork)
						destPath := utils.SlashJoin(cd, gwHome, serverDir, receive.InPath)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule only has a 'work' directory", func() {
						receive := &model.Rule{
							Name:     "receive",
							IsSend:   false,
							Path:     "path",
							WorkPath: "rule_work",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverDir, receive.WorkPath)
						destPath := utils.SlashJoin(cd, gwHome, serverDir)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule has neither an 'in' or a 'work' directory", func() {
						receive := &model.Rule{
							Name:   "receive",
							IsSend: false,
							Path:   "path",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverWork)
						destPath := utils.SlashJoin(cd, gwHome, serverDir)
						testFunc(receive.ID, workPath, destPath)
					})
				})

				Convey("Given a server with only a root directory", func() {
					serverDir := "server_root"
					serverWork := serverDir
					paths.ServerRoot = utils.SlashJoin(gwRoot, serverDir)
					paths.ServerWork = utils.SlashJoin(gwRoot, serverWork)

					Convey("Given that the rule has both an 'in' and 'work' directory", func() {
						receive := &model.Rule{
							Name:     "receive",
							IsSend:   false,
							Path:     "path",
							InPath:   "rule_in",
							WorkPath: "rule_work",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverDir, receive.WorkPath)
						destPath := utils.SlashJoin(cd, gwHome, serverDir, receive.InPath)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule only has an 'in' directory", func() {
						receive := &model.Rule{
							Name:   "receive",
							IsSend: false,
							Path:   "path",
							InPath: "rule_in",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverWork)
						destPath := utils.SlashJoin(cd, gwHome, serverDir, receive.InPath)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule only has a 'work' directory", func() {
						receive := &model.Rule{
							Name:     "receive",
							IsSend:   false,
							Path:     "path",
							WorkPath: "rule_work",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverDir, receive.WorkPath)
						destPath := utils.SlashJoin(cd, gwHome, serverDir)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule has neither an 'in' or a 'work' directory", func() {
						receive := &model.Rule{
							Name:   "receive",
							IsSend: false,
							Path:   "path",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverWork)
						destPath := utils.SlashJoin(cd, gwHome, serverDir)
						testFunc(receive.ID, workPath, destPath)
					})
				})

				Convey("Given a server with only a work directory", func() {
					serverWork := "server_root/server_work"
					paths.ServerWork = utils.SlashJoin(gwRoot, serverWork)

					Convey("Given that the rule has both an 'in' and 'work' directory", func() {
						receive := &model.Rule{
							Name:     "receive",
							IsSend:   false,
							Path:     "path",
							InPath:   "rule_in",
							WorkPath: "rule_work",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, receive.WorkPath)
						destPath := utils.SlashJoin(cd, gwHome, receive.InPath)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule only has an 'in' directory", func() {
						receive := &model.Rule{
							Name:   "receive",
							IsSend: false,
							Path:   "path",
							InPath: "rule_in",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverWork)
						destPath := utils.SlashJoin(cd, gwHome, receive.InPath)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule only has a 'work' directory", func() {
						receive := &model.Rule{
							Name:     "receive",
							IsSend:   false,
							Path:     "path",
							WorkPath: "rule_work",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, receive.WorkPath)
						destPath := utils.SlashJoin(cd, gwHome, inDir)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule has neither an 'in' or a 'work' directory", func() {
						receive := &model.Rule{
							Name:   "receive",
							IsSend: false,
							Path:   "path",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverWork)
						destPath := utils.SlashJoin(cd, gwHome, inDir)
						testFunc(receive.ID, workPath, destPath)
					})
				})

				Convey("Given a server with neither a root or work directory", func() {

					Convey("Given that the rule has both an 'in' and 'work' directory", func() {
						receive := &model.Rule{
							Name:     "receive",
							IsSend:   false,
							Path:     "path",
							InPath:   "rule_in",
							WorkPath: "rule_work",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, receive.WorkPath)
						destPath := utils.SlashJoin(cd, gwHome, receive.InPath)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule only has an 'in' directory", func() {
						receive := &model.Rule{
							Name:   "receive",
							IsSend: false,
							Path:   "path",
							InPath: "rule_in",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, workDir)
						destPath := utils.SlashJoin(cd, gwHome, receive.InPath)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule only has a 'work' directory", func() {
						receive := &model.Rule{
							Name:     "receive",
							IsSend:   false,
							Path:     "path",
							WorkPath: "rule_work",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, receive.WorkPath)
						destPath := utils.SlashJoin(cd, gwHome, inDir)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule has neither an 'in' or a 'work' directory", func() {
						receive := &model.Rule{
							Name:   "receive",
							IsSend: false,
							Path:   "path",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, workDir)
						destPath := utils.SlashJoin(cd, gwHome, inDir)
						testFunc(receive.ID, workPath, destPath)
					})
				})
			})

			Convey("Given that it has only an 'in' directory", func() {
				inDir := "in"
				paths.InDirectory = utils.SlashJoin(gwRoot, inDir)
				paths.WorkDirectory = gwRoot

				Convey("Given a server with a root & work directory", func() {
					serverDir := "server_root"
					serverWork := serverDir + "/server_work"
					paths.ServerRoot = utils.SlashJoin(gwRoot, serverDir)
					paths.ServerWork = utils.SlashJoin(gwRoot, serverWork)

					Convey("Given that the rule has both an 'in' and 'work' directory", func() {
						receive := &model.Rule{
							Name:     "receive",
							IsSend:   false,
							Path:     "path",
							InPath:   "rule_in",
							WorkPath: "rule_work",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverDir, receive.WorkPath)
						destPath := utils.SlashJoin(cd, gwHome, serverDir, receive.InPath)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule only has an 'in' directory", func() {
						receive := &model.Rule{
							Name:   "receive",
							IsSend: false,
							Path:   "path",
							InPath: "rule_in",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverWork)
						destPath := utils.SlashJoin(cd, gwHome, serverDir, receive.InPath)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule only has a 'work' directory", func() {
						receive := &model.Rule{
							Name:     "receive",
							IsSend:   false,
							Path:     "path",
							WorkPath: "rule_work",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverDir, receive.WorkPath)
						destPath := utils.SlashJoin(cd, gwHome, serverDir)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule has neither an 'in' or a 'work' directory", func() {
						receive := &model.Rule{
							Name:   "receive",
							IsSend: false,
							Path:   "path",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverWork)
						destPath := utils.SlashJoin(cd, gwHome, serverDir)
						testFunc(receive.ID, workPath, destPath)
					})
				})

				Convey("Given a server with only a root directory", func() {
					serverDir := "server_root"
					serverWork := serverDir
					paths.ServerRoot = utils.SlashJoin(gwRoot, serverDir)
					paths.ServerWork = utils.SlashJoin(gwRoot, serverWork)

					Convey("Given that the rule has both an 'in' and 'work' directory", func() {
						receive := &model.Rule{
							Name:     "receive",
							IsSend:   false,
							Path:     "path",
							InPath:   "rule_in",
							WorkPath: "rule_work",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverDir, receive.WorkPath)
						destPath := utils.SlashJoin(cd, gwHome, serverDir, receive.InPath)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule only has an 'in' directory", func() {
						receive := &model.Rule{
							Name:   "receive",
							IsSend: false,
							Path:   "path",
							InPath: "rule_in",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverWork)
						destPath := utils.SlashJoin(cd, gwHome, serverDir, receive.InPath)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule only has a 'work' directory", func() {
						receive := &model.Rule{
							Name:     "receive",
							IsSend:   false,
							Path:     "path",
							WorkPath: "rule_work",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverDir, receive.WorkPath)
						destPath := utils.SlashJoin(cd, gwHome, serverDir)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule has neither an 'in' or a 'work' directory", func() {
						receive := &model.Rule{
							Name:   "receive",
							IsSend: false,
							Path:   "path",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverWork)
						destPath := utils.SlashJoin(cd, gwHome, serverDir)
						testFunc(receive.ID, workPath, destPath)
					})
				})

				Convey("Given a server with only a work directory", func() {
					serverWork := "server_root/server_work"
					paths.ServerWork = utils.SlashJoin(gwRoot, serverWork)

					Convey("Given that the rule has both an 'in' and 'work' directory", func() {
						receive := &model.Rule{
							Name:     "receive",
							IsSend:   false,
							Path:     "path",
							InPath:   "rule_in",
							WorkPath: "rule_work",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, receive.WorkPath)
						destPath := utils.SlashJoin(cd, gwHome, receive.InPath)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule only has an 'in' directory", func() {
						receive := &model.Rule{
							Name:   "receive",
							IsSend: false,
							Path:   "path",
							InPath: "rule_in",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverWork)
						destPath := utils.SlashJoin(cd, gwHome, receive.InPath)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule only has a 'work' directory", func() {
						receive := &model.Rule{
							Name:     "receive",
							IsSend:   false,
							Path:     "path",
							WorkPath: "rule_work",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, receive.WorkPath)
						destPath := utils.SlashJoin(cd, gwHome, inDir)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule has neither an 'in' or a 'work' directory", func() {
						receive := &model.Rule{
							Name:   "receive",
							IsSend: false,
							Path:   "path",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverWork)
						destPath := utils.SlashJoin(cd, gwHome, inDir)
						testFunc(receive.ID, workPath, destPath)
					})
				})

				Convey("Given a server with neither a root or work directory", func() {

					Convey("Given that the rule has both an 'in' and 'work' directory", func() {
						receive := &model.Rule{
							Name:     "receive",
							IsSend:   false,
							Path:     "path",
							InPath:   "rule_in",
							WorkPath: "rule_work",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, receive.WorkPath)
						destPath := utils.SlashJoin(cd, gwHome, receive.InPath)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule only has an 'in' directory", func() {
						receive := &model.Rule{
							Name:   "receive",
							IsSend: false,
							Path:   "path",
							InPath: "rule_in",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome)
						destPath := utils.SlashJoin(cd, gwHome, receive.InPath)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule only has a 'work' directory", func() {
						receive := &model.Rule{
							Name:     "receive",
							IsSend:   false,
							Path:     "path",
							WorkPath: "rule_work",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, receive.WorkPath)
						destPath := utils.SlashJoin(cd, gwHome, inDir)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule has neither an 'in' or a 'work' directory", func() {
						receive := &model.Rule{
							Name:   "receive",
							IsSend: false,
							Path:   "path",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome)
						destPath := utils.SlashJoin(cd, gwHome, inDir)
						testFunc(receive.ID, workPath, destPath)
					})
				})
			})

			Convey("Given that it has only a 'work' directory", func() {
				workDir := "tmp"
				paths.InDirectory = gwRoot
				paths.WorkDirectory = utils.SlashJoin(gwRoot, workDir)

				Convey("Given a server with a root & work directory", func() {
					serverDir := "server_root"
					serverWork := serverDir + "/server_work"
					paths.ServerRoot = utils.SlashJoin(gwRoot, serverDir)
					paths.ServerWork = utils.SlashJoin(gwRoot, serverWork)

					Convey("Given that the rule has both an 'in' and 'work' directory", func() {
						receive := &model.Rule{
							Name:     "receive",
							IsSend:   false,
							Path:     "path",
							InPath:   "rule_in",
							WorkPath: "rule_work",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverDir, receive.WorkPath)
						destPath := utils.SlashJoin(cd, gwHome, serverDir, receive.InPath)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule only has an 'in' directory", func() {
						receive := &model.Rule{
							Name:   "receive",
							IsSend: false,
							Path:   "path",
							InPath: "rule_in",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverWork)
						destPath := utils.SlashJoin(cd, gwHome, serverDir, receive.InPath)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule only has a 'work' directory", func() {
						receive := &model.Rule{
							Name:     "receive",
							IsSend:   false,
							Path:     "path",
							WorkPath: "rule_work",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverDir, receive.WorkPath)
						destPath := utils.SlashJoin(cd, gwHome, serverDir)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule has neither an 'in' or a 'work' directory", func() {
						receive := &model.Rule{
							Name:   "receive",
							IsSend: false,
							Path:   "path",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverWork)
						destPath := utils.SlashJoin(cd, gwHome, serverDir)
						testFunc(receive.ID, workPath, destPath)
					})
				})

				Convey("Given a server with only a root directory", func() {
					serverDir := "server_root"
					serverWork := serverDir
					paths.ServerRoot = utils.SlashJoin(gwRoot, serverDir)
					paths.ServerWork = utils.SlashJoin(gwRoot, serverWork)

					Convey("Given that the rule has both an 'in' and 'work' directory", func() {
						receive := &model.Rule{
							Name:     "receive",
							IsSend:   false,
							Path:     "path",
							InPath:   "rule_in",
							WorkPath: "rule_work",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverDir, receive.WorkPath)
						destPath := utils.SlashJoin(cd, gwHome, serverDir, receive.InPath)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule only has an 'in' directory", func() {
						receive := &model.Rule{
							Name:   "receive",
							IsSend: false,
							Path:   "path",
							InPath: "rule_in",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverWork)
						destPath := utils.SlashJoin(cd, gwHome, serverDir, receive.InPath)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule only has a 'work' directory", func() {
						receive := &model.Rule{
							Name:     "receive",
							IsSend:   false,
							Path:     "path",
							WorkPath: "rule_work",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverDir, receive.WorkPath)
						destPath := utils.SlashJoin(cd, gwHome, serverDir)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule has neither an 'in' or a 'work' directory", func() {
						receive := &model.Rule{
							Name:   "receive",
							IsSend: false,
							Path:   "path",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverWork)
						destPath := utils.SlashJoin(cd, gwHome, serverDir)
						testFunc(receive.ID, workPath, destPath)
					})
				})

				Convey("Given a server with only a work directory", func() {
					serverWork := "server_root/server_work"
					paths.ServerWork = utils.SlashJoin(gwRoot, serverWork)

					Convey("Given that the rule has both an 'in' and 'work' directory", func() {
						receive := &model.Rule{
							Name:     "receive",
							IsSend:   false,
							Path:     "path",
							InPath:   "rule_in",
							WorkPath: "rule_work",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, receive.WorkPath)
						destPath := utils.SlashJoin(cd, gwHome, receive.InPath)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule only has an 'in' directory", func() {
						receive := &model.Rule{
							Name:   "receive",
							IsSend: false,
							Path:   "path",
							InPath: "rule_in",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverWork)
						destPath := utils.SlashJoin(cd, gwHome, receive.InPath)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule only has a 'work' directory", func() {
						receive := &model.Rule{
							Name:     "receive",
							IsSend:   false,
							Path:     "path",
							WorkPath: "rule_work",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, receive.WorkPath)
						destPath := utils.SlashJoin(cd, gwHome)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule has neither an 'in' or a 'work' directory", func() {
						receive := &model.Rule{
							Name:   "receive",
							IsSend: false,
							Path:   "path",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverWork)
						destPath := utils.SlashJoin(cd, gwHome)
						testFunc(receive.ID, workPath, destPath)
					})
				})

				Convey("Given a server with neither a root or work directory", func() {

					Convey("Given that the rule has both an 'in' and 'work' directory", func() {
						receive := &model.Rule{
							Name:     "receive",
							IsSend:   false,
							Path:     "path",
							InPath:   "rule_in",
							WorkPath: "rule_work",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, receive.WorkPath)
						destPath := utils.SlashJoin(cd, gwHome, receive.InPath)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule only has an 'in' directory", func() {
						receive := &model.Rule{
							Name:   "receive",
							IsSend: false,
							Path:   "path",
							InPath: "rule_in",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, workDir)
						destPath := utils.SlashJoin(cd, gwHome, receive.InPath)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule only has a 'work' directory", func() {
						receive := &model.Rule{
							Name:     "receive",
							IsSend:   false,
							Path:     "path",
							WorkPath: "rule_work",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, receive.WorkPath)
						destPath := utils.SlashJoin(cd, gwHome)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule has neither an 'in' or a 'work' directory", func() {
						receive := &model.Rule{
							Name:   "receive",
							IsSend: false,
							Path:   "path",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, workDir)
						destPath := utils.SlashJoin(cd, gwHome)
						testFunc(receive.ID, workPath, destPath)
					})
				})
			})

			Convey("Given that it has neither a 'in' or 'work' directory", func() {
				paths.InDirectory = gwRoot
				paths.WorkDirectory = gwRoot

				Convey("Given a server with a root & work directory", func() {
					serverDir := "server_root"
					serverWork := serverDir + "/server_work"
					paths.ServerRoot = utils.SlashJoin(gwRoot, serverDir)
					paths.ServerWork = utils.SlashJoin(gwRoot, serverWork)

					Convey("Given that the rule has both an 'in' and 'work' directory", func() {
						receive := &model.Rule{
							Name:     "receive",
							IsSend:   false,
							Path:     "path",
							InPath:   "rule_in",
							WorkPath: "rule_work",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverDir, receive.WorkPath)
						destPath := utils.SlashJoin(cd, gwHome, serverDir, receive.InPath)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule only has an 'in' directory", func() {
						receive := &model.Rule{
							Name:   "receive",
							IsSend: false,
							Path:   "path",
							InPath: "rule_in",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverWork)
						destPath := utils.SlashJoin(cd, gwHome, serverDir, receive.InPath)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule only has a 'work' directory", func() {
						receive := &model.Rule{
							Name:     "receive",
							IsSend:   false,
							Path:     "path",
							WorkPath: "rule_work",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverDir, receive.WorkPath)
						destPath := utils.SlashJoin(cd, gwHome, serverDir)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule has neither an 'in' or a 'work' directory", func() {
						receive := &model.Rule{
							Name:   "receive",
							IsSend: false,
							Path:   "path",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverWork)
						destPath := utils.SlashJoin(cd, gwHome, serverDir)
						testFunc(receive.ID, workPath, destPath)
					})
				})

				Convey("Given a server with only a root directory", func() {
					serverDir := "server_root"
					serverWork := serverDir
					paths.ServerRoot = utils.SlashJoin(gwRoot, serverDir)
					paths.ServerWork = utils.SlashJoin(gwRoot, serverWork)

					Convey("Given that the rule has both an 'in' and 'work' directory", func() {
						receive := &model.Rule{
							Name:     "receive",
							IsSend:   false,
							Path:     "path",
							InPath:   "rule_in",
							WorkPath: "rule_work",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverDir, receive.WorkPath)
						destPath := utils.SlashJoin(cd, gwHome, serverDir, receive.InPath)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule only has an 'in' directory", func() {
						receive := &model.Rule{
							Name:   "receive",
							IsSend: false,
							Path:   "path",
							InPath: "rule_in",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverWork)
						destPath := utils.SlashJoin(cd, gwHome, serverDir, receive.InPath)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule only has a 'work' directory", func() {
						receive := &model.Rule{
							Name:     "receive",
							IsSend:   false,
							Path:     "path",
							WorkPath: "rule_work",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverDir, receive.WorkPath)
						destPath := utils.SlashJoin(cd, gwHome, serverDir)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule has neither an 'in' or a 'work' directory", func() {
						receive := &model.Rule{
							Name:   "receive",
							IsSend: false,
							Path:   "path",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverWork)
						destPath := utils.SlashJoin(cd, gwHome, serverDir)
						testFunc(receive.ID, workPath, destPath)
					})
				})

				Convey("Given a server with only a work directory", func() {
					serverWork := "server_root/server_work"
					paths.ServerWork = utils.SlashJoin(gwRoot, serverWork)

					Convey("Given that the rule has both an 'in' and 'work' directory", func() {
						receive := &model.Rule{
							Name:     "receive",
							IsSend:   false,
							Path:     "path",
							InPath:   "rule_in",
							WorkPath: "rule_work",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, receive.WorkPath)
						destPath := utils.SlashJoin(cd, gwHome, receive.InPath)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule only has an 'in' directory", func() {
						receive := &model.Rule{
							Name:   "receive",
							IsSend: false,
							Path:   "path",
							InPath: "rule_in",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverWork)
						destPath := utils.SlashJoin(cd, gwHome, receive.InPath)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule only has a 'work' directory", func() {
						receive := &model.Rule{
							Name:     "receive",
							IsSend:   false,
							Path:     "path",
							WorkPath: "rule_work",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, receive.WorkPath)
						destPath := utils.SlashJoin(cd, gwHome)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule has neither an 'in' or a 'work' directory", func() {
						receive := &model.Rule{
							Name:   "receive",
							IsSend: false,
							Path:   "path",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, serverWork)
						destPath := utils.SlashJoin(cd, gwHome)
						testFunc(receive.ID, workPath, destPath)
					})
				})

				Convey("Given a server with neither a root or work directory", func() {

					Convey("Given that the rule has both an 'in' and 'work' directory", func() {
						receive := &model.Rule{
							Name:     "receive",
							IsSend:   false,
							Path:     "path",
							InPath:   "rule_in",
							WorkPath: "rule_work",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, receive.WorkPath)
						destPath := utils.SlashJoin(cd, gwHome, receive.InPath)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule only has an 'in' directory", func() {
						receive := &model.Rule{
							Name:   "receive",
							IsSend: false,
							Path:   "path",
							InPath: "rule_in",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome)
						destPath := utils.SlashJoin(cd, gwHome, receive.InPath)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule only has a 'work' directory", func() {
						receive := &model.Rule{
							Name:     "receive",
							IsSend:   false,
							Path:     "path",
							WorkPath: "rule_work",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome, receive.WorkPath)
						destPath := utils.SlashJoin(cd, gwHome)
						testFunc(receive.ID, workPath, destPath)
					})

					Convey("Given that the rule has neither an 'in' or a 'work' directory", func() {
						receive := &model.Rule{
							Name:   "receive",
							IsSend: false,
							Path:   "path",
						}
						So(db.Create(receive), ShouldBeNil)

						workPath := utils.SlashJoin(cd, gwHome)
						destPath := utils.SlashJoin(cd, gwHome)
						testFunc(receive.ID, workPath, destPath)
					})
				})
			})
		})
	})
}

func TestPathOut(t *testing.T) {
	logger := log.NewLogger("test_path_out")

	gwHome := "out_path_test_root"

	cd, err := os.Getwd()
	if err != nil {
		t.FailNow()
	}

	gwRoot := utils.SlashJoin(cd, gwHome)

	Convey("Given a Gateway configuration", t, func() {
		paths := Paths{
			PathsConfig: conf.PathsConfig{GatewayHome: gwRoot},
		}
		Reset(func() { _ = os.RemoveAll(gwRoot) })

		Convey("Given some transfer agents", func() {
			db := database.GetTestDatabase()

			localAgent := &model.LocalAgent{
				Name:        "local_agent",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(localAgent), ShouldBeNil)

			localAccount := &model.LocalAccount{
				LocalAgentID: localAgent.ID,
				Login:        "local_account",
				Password:     []byte("password"),
			}
			So(db.Create(localAccount), ShouldBeNil)

			remoteAgent := &model.RemoteAgent{
				Name:        "remote_agent",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(remoteAgent), ShouldBeNil)

			remoteAccount := &model.RemoteAccount{
				RemoteAgentID: remoteAgent.ID,
				Login:         "remote_account",
				Password:      []byte("password"),
			}
			So(db.Create(remoteAccount), ShouldBeNil)

			testFunc := func(ruleID uint64, srcPath string) {
				Convey("When creating & starting a transfer stream", func() {
					trans := model.Transfer{
						RuleID:     ruleID,
						IsServer:   true,
						AgentID:    localAgent.ID,
						AccountID:  localAccount.ID,
						SourceFile: "file.src",
						DestFile:   "file.dst",
					}
					path := utils.SlashJoin(srcPath, trans.SourceFile)
					So(os.MkdirAll(srcPath, 0700), ShouldBeNil)
					So(ioutil.WriteFile(path, nil, 0700), ShouldBeNil)

					stream, err := NewTransferStream(context.Background(),
						logger, db, paths, trans)
					So(err, ShouldBeNil)

					err = stream.Start()
					So(err, ShouldBeNil)

					Convey("Then it should have opened the correct source file", func() {
						So(stream.Name(), ShouldEqual, path)
					})
				})
			}

			Convey("Given that it has an 'out' directory", func() {
				outDir := "out"
				paths.OutDirectory = utils.SlashJoin(gwRoot, outDir)

				Convey("Given a server with a root directory", func() {
					serverDir := "server_root"
					paths.ServerRoot = utils.SlashJoin(gwRoot, serverDir)

					Convey("Given that the rule has an 'out' directory", func() {
						send := &model.Rule{
							Name:    "send",
							IsSend:  true,
							Path:    "path",
							OutPath: "rule_out",
						}
						So(db.Create(send), ShouldBeNil)

						outPath := utils.SlashJoin(cd, gwHome, serverDir, send.OutPath)
						testFunc(send.ID, outPath)
					})

					Convey("Given that the rule does not have an 'out' directory", func() {
						send := &model.Rule{
							Name:   "send",
							IsSend: true,
							Path:   "path",
						}
						So(db.Create(send), ShouldBeNil)

						outPath := utils.SlashJoin(cd, gwHome, serverDir)
						testFunc(send.ID, outPath)
					})
				})

				Convey("Given a server without a root directory", func() {

					Convey("Given that the rule has an 'out' directory", func() {
						send := &model.Rule{
							Name:    "send",
							IsSend:  true,
							Path:    "path",
							OutPath: "rule_out",
						}
						So(db.Create(send), ShouldBeNil)

						outPath := utils.SlashJoin(cd, gwHome, send.OutPath)
						testFunc(send.ID, outPath)
					})

					Convey("Given that the rule does not have an 'out' directory", func() {
						send := &model.Rule{
							Name:   "send",
							IsSend: true,
							Path:   "path",
						}
						So(db.Create(send), ShouldBeNil)

						outPath := utils.SlashJoin(cd, gwHome, outDir)
						testFunc(send.ID, outPath)
					})
				})
			})

			Convey("Given that it does not have an 'out' directory", func() {
				paths.OutDirectory = gwRoot

				Convey("Given a server with a root directory", func() {
					serverDir := "server_root"
					paths.ServerRoot = utils.SlashJoin(gwRoot, serverDir)

					Convey("Given that the rule has an 'out' directory", func() {
						send := &model.Rule{
							Name:    "send",
							IsSend:  true,
							Path:    "path",
							OutPath: "rule_out",
						}
						So(db.Create(send), ShouldBeNil)

						outPath := utils.SlashJoin(cd, gwHome, serverDir, send.OutPath)
						testFunc(send.ID, outPath)
					})

					Convey("Given that the rule does not have an 'out' directory", func() {
						send := &model.Rule{
							Name:   "send",
							IsSend: true,
							Path:   "path",
						}
						So(db.Create(send), ShouldBeNil)

						outPath := utils.SlashJoin(cd, gwHome, serverDir)
						testFunc(send.ID, outPath)
					})
				})

				Convey("Given a server without a root directory", func() {

					Convey("Given that the rule has an 'out' directory", func() {
						send := &model.Rule{
							Name:    "send",
							IsSend:  true,
							Path:    "path",
							OutPath: "rule_out",
						}
						So(db.Create(send), ShouldBeNil)

						outPath := utils.SlashJoin(cd, gwHome, send.OutPath)
						testFunc(send.ID, outPath)
					})

					Convey("Given that the rule does not have an 'out' directory", func() {
						send := &model.Rule{
							Name:   "send",
							IsSend: true,
							Path:   "path",
						}
						So(db.Create(send), ShouldBeNil)

						outPath := utils.SlashJoin(cd, gwHome)
						testFunc(send.ID, outPath)
					})
				})
			})
		})
	})
}
