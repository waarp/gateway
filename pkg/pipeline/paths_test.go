package pipeline

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	. "path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

func TestPathIn(t *testing.T) {
	logger := log.NewLogger("test_path_in")

	gwHome := "in_path_test_root"

	cd, err := os.Getwd()
	if err != nil {
		t.FailNow()
	}

	gwRoot := Join(cd, gwHome)

	Convey("Given a Gateway configuration", t, func() {
		paths := Paths{
			PathsConfig: conf.PathsConfig{
				GatewayHome:   gwRoot,
				InDirectory:   "gwIn",
				OutDirectory:  "gwOut",
				WorkDirectory: "gwWork",
			},
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

			receive := &model.Rule{
				Name:   "receive",
				IsSend: false,
				Path:   "receive_path",
			}
			So(db.Create(receive), ShouldBeNil)

			trans := model.Transfer{
				RuleID:     receive.ID,
				IsServer:   true,
				AgentID:    localAgent.ID,
				AccountID:  localAccount.ID,
				SourceFile: "file.src",
				DestFile:   "file.dst",
			}

			file := trans.DestFile
			tmp := trans.DestFile + ".tmp"

			testCases := []struct {
				serRoot, ruleIn, ruleWork string
				expTmp, expFinal          string
			}{
				{"", "", "", Join(gwRoot, "gwWork", tmp), Join(gwRoot, "gwIn", file)},
				{"serRoot", "", "", Join(gwRoot, "serRoot", "serWork", tmp), Join(gwRoot, "serRoot", "serIn", file)},
				{"", "ruleIn", "", Join(gwRoot, "gwWork", tmp), Join(gwRoot, "ruleIn", file)},
				{"", "", "ruleWork", Join(gwRoot, "ruleWork", tmp), Join(gwRoot, "gwIn", file)},
				{"serRoot", "ruleIn", "", Join(gwRoot, "serRoot", "serWork", tmp), Join(gwRoot, "serRoot", "ruleIn", file)},
				{"serRoot", "", "ruleWork", Join(gwRoot, "serRoot", "ruleWork", tmp), Join(gwRoot, "serRoot", "serIn", file)},
				{"", "ruleIn", "ruleWork", Join(gwRoot, "ruleWork", tmp), Join(gwRoot, "ruleIn", file)},
				{"serRoot", "ruleIn", "ruleWork", Join(gwRoot, "serRoot", "ruleWork", tmp), Join(gwRoot, "serRoot", "ruleIn", file)},
			}

			for _, tc := range testCases {
				Convey(fmt.Sprintf("Given the following path parameters: %v", tc), func() {
					paths.ServerRoot = tc.serRoot
					if paths.ServerRoot != "" {
						paths.ServerIn = "serIn"
						paths.ServerOut = "serOut"
						paths.ServerWork = "serWork"
					} else {
						paths.ServerIn = ""
						paths.ServerOut = ""
						paths.ServerWork = ""
					}

					rule := "UPDATE rules SET in_path=?, work_path=? WHERE id=?"
					So(db.Execute(rule, tc.ruleIn, tc.ruleWork, receive.ID), ShouldBeNil)

					Convey("When launching a transfer stream", func() {
						stream, err := NewTransferStream(context.Background(),
							logger, db, paths, trans)
						So(err, ShouldBeNil)
						Reset(func() { _ = stream.Close() })

						So(stream.Start(), ShouldBeNil)

						Convey("Then it should have created the correct work file", func() {
							_, err := os.Stat(tc.expTmp)
							So(err, ShouldBeNil)
						})

						Convey("When finalizing the transfer", func() {
							So(stream.Close(), ShouldBeNil)

							Convey("Then it should have moved the file to its destination", func() {
								_, err := os.Stat(tc.expFinal)
								So(err, ShouldBeNil)
							})
						})
					})
				})
			}
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

	gwRoot := Join(cd, gwHome)

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
					path := Join(srcPath, trans.SourceFile)
					So(os.MkdirAll(srcPath, 0o700), ShouldBeNil)
					So(ioutil.WriteFile(path, nil, 0o700), ShouldBeNil)

					stream, err := NewTransferStream(context.Background(),
						logger, db, paths, trans)
					So(err, ShouldBeNil)
					Reset(func() { _ = stream.Close() })

					err = stream.Start()
					So(err, ShouldBeNil)

					Convey("Then it should have opened the correct source file", func() {
						So(stream.Name(), ShouldEqual, path)
					})
				})
			}

			Convey("Given that it has an 'out' directory", func() {
				outDir := "out"
				paths.OutDirectory = Join(gwRoot, outDir)

				Convey("Given a server with a root directory", func() {
					serverDir := "server_root"
					paths.ServerRoot = Join(gwRoot, serverDir)

					Convey("Given that the rule has an 'out' directory", func() {
						send := &model.Rule{
							Name:    "send",
							IsSend:  true,
							Path:    "path",
							OutPath: "rule_out",
						}
						So(db.Create(send), ShouldBeNil)

						outPath := Join(cd, gwHome, serverDir, send.OutPath)
						testFunc(send.ID, outPath)
					})

					Convey("Given that the rule does not have an 'out' directory", func() {
						send := &model.Rule{
							Name:   "send",
							IsSend: true,
							Path:   "path",
						}
						So(db.Create(send), ShouldBeNil)

						outPath := Join(cd, gwHome, serverDir)
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

						outPath := Join(cd, gwHome, send.OutPath)
						testFunc(send.ID, outPath)
					})

					Convey("Given that the rule does not have an 'out' directory", func() {
						send := &model.Rule{
							Name:   "send",
							IsSend: true,
							Path:   "path",
						}
						So(db.Create(send), ShouldBeNil)

						outPath := Join(cd, gwHome, outDir)
						testFunc(send.ID, outPath)
					})
				})
			})

			Convey("Given that it does not have an 'out' directory", func() {
				paths.OutDirectory = gwRoot

				Convey("Given a server with a root directory", func() {
					serverDir := "server_root"
					paths.ServerRoot = Join(gwRoot, serverDir)

					Convey("Given that the rule has an 'out' directory", func() {
						send := &model.Rule{
							Name:    "send",
							IsSend:  true,
							Path:    "path",
							OutPath: "rule_out",
						}
						So(db.Create(send), ShouldBeNil)

						outPath := Join(cd, gwHome, serverDir, send.OutPath)
						testFunc(send.ID, outPath)
					})

					Convey("Given that the rule does not have an 'out' directory", func() {
						send := &model.Rule{
							Name:   "send",
							IsSend: true,
							Path:   "path",
						}
						So(db.Create(send), ShouldBeNil)

						outPath := Join(cd, gwHome, serverDir)
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

						outPath := Join(cd, gwHome, send.OutPath)
						testFunc(send.ID, outPath)
					})

					Convey("Given that the rule does not have an 'out' directory", func() {
						send := &model.Rule{
							Name:   "send",
							IsSend: true,
							Path:   "path",
						}
						So(db.Create(send), ShouldBeNil)

						outPath := Join(cd, gwHome)
						testFunc(send.ID, outPath)
					})
				})
			})
		})
	})
}
