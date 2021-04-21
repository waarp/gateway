package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	. "path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"
)

func TestPathIn(t *testing.T) {
	logger := log.NewLogger("test_path_in")

	Convey("Given a Gateway configuration", t, func(c C) {
		gwRoot := testhelpers.TempDir(c, "foo")
		paths := Paths{
			PathsConfig: conf.PathsConfig{
				GatewayHome:   gwRoot,
				InDirectory:   filepath.Join(gwRoot, "gwIn"),
				OutDirectory:  filepath.Join(gwRoot, "gwOut"),
				WorkDirectory: filepath.Join(gwRoot, "gwWork"),
			},
		}

		Convey("Given some transfer agents", func(c C) {
			db := database.TestDatabase(c, "ERROR")

			localAgent := &model.LocalAgent{
				Name:        "local_agent",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1111",
			}
			So(db.Insert(localAgent).Run(), ShouldBeNil)

			localAccount := &model.LocalAccount{
				LocalAgentID: localAgent.ID,
				Login:        "local_account",
				PasswordHash: hash("password"),
			}
			So(db.Insert(localAccount).Run(), ShouldBeNil)

			remoteAgent := &model.RemoteAgent{
				Name:        "remote_agent",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2222",
			}
			So(db.Insert(remoteAgent).Run(), ShouldBeNil)

			remoteAccount := &model.RemoteAccount{
				RemoteAgentID: remoteAgent.ID,
				Login:         "remote_account",
				Password:      "password",
			}
			So(db.Insert(remoteAccount).Run(), ShouldBeNil)

			receive := &model.Rule{
				Name:   "receive",
				IsSend: false,
				Path:   "receive_path",
			}
			So(db.Insert(receive).Run(), ShouldBeNil)

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

			type testCase struct {
				serRoot, ruleIn, ruleWork string
				expTmp, expFinal          string
			}
			testCases := []testCase{
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
				testCaseName := fmt.Sprintf(
					"Given the following path parameters: %q %q %q",
					tc.serRoot, tc.ruleIn, tc.ruleWork,
				)
				Convey(testCaseName, func() {
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

					rule := &model.Rule{
						ID:       receive.ID,
						Name:     receive.Name,
						InPath:   tc.ruleIn,
						WorkPath: tc.ruleWork,
					}
					So(db.Update(rule).Cols("in_path", "work_path").Run(), ShouldBeNil)

					Convey("When launching a transfer stream", func() {
						stream, err := NewTransferStream(context.Background(),
							logger, db, paths, &trans)
						So(err, ShouldBeNil)

						So(stream.Start(), ShouldBeNil)

						Convey("Then it should have created the correct work file", func() {
							Reset(func() { So(stream.Close(), ShouldBeNil) })
							_, err := os.Stat(tc.expTmp)
							So(err, ShouldBeNil)
						})

						Convey("When finalizing the transfer", func() {
							So(stream.Close(), ShouldBeNil)
							So(stream.Move(), ShouldBeNil)

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

	Convey("Given a Gateway configuration", t, func(c C) {
		gwRoot := testhelpers.TempDir(c, "out_path_root")
		paths := Paths{
			PathsConfig: conf.PathsConfig{GatewayHome: gwRoot},
		}

		Convey("Given some transfer agents", func(c C) {
			db := database.TestDatabase(c, "ERROR")

			localAgent := &model.LocalAgent{
				Name:        "local_agent",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1111",
			}
			So(db.Insert(localAgent).Run(), ShouldBeNil)

			localAccount := &model.LocalAccount{
				LocalAgentID: localAgent.ID,
				Login:        "local_account",
				PasswordHash: hash("password"),
			}
			So(db.Insert(localAccount).Run(), ShouldBeNil)
			So(db.Insert(localAccount).Run(), ShouldBeNil)

			remoteAgent := &model.RemoteAgent{
				Name:        "remote_agent",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2222",
			}
			So(db.Insert(remoteAgent).Run(), ShouldBeNil)

			remoteAccount := &model.RemoteAccount{
				RemoteAgentID: remoteAgent.ID,
				Login:         "remote_account",
				Password:      "password",
			}
			So(db.Insert(remoteAccount).Run(), ShouldBeNil)

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
						logger, db, paths, &trans)
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
						So(db.Insert(send).Run(), ShouldBeNil)

						outPath := Join(gwRoot, serverDir, send.OutPath)
						testFunc(send.ID, outPath)
					})

					Convey("Given that the rule does not have an 'out' directory", func() {
						send := &model.Rule{
							Name:   "send",
							IsSend: true,
							Path:   "path",
						}
						So(db.Insert(send).Run(), ShouldBeNil)

						outPath := Join(gwRoot, serverDir)
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
						So(db.Insert(send).Run(), ShouldBeNil)

						outPath := Join(gwRoot, send.OutPath)
						testFunc(send.ID, outPath)
					})

					Convey("Given that the rule does not have an 'out' directory", func() {
						send := &model.Rule{
							Name:   "send",
							IsSend: true,
							Path:   "path",
						}
						So(db.Insert(send).Run(), ShouldBeNil)

						outPath := Join(gwRoot, outDir)
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
						So(db.Insert(send).Run(), ShouldBeNil)

						outPath := Join(gwRoot, serverDir, send.OutPath)
						testFunc(send.ID, outPath)
					})

					Convey("Given that the rule does not have an 'out' directory", func() {
						send := &model.Rule{
							Name:   "send",
							IsSend: true,
							Path:   "path",
						}
						So(db.Insert(send).Run(), ShouldBeNil)

						outPath := Join(gwRoot, serverDir)
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
						So(db.Insert(send).Run(), ShouldBeNil)

						outPath := Join(gwRoot, send.OutPath)
						testFunc(send.ID, outPath)
					})

					Convey("Given that the rule does not have an 'out' directory", func() {
						send := &model.Rule{
							Name:   "send",
							IsSend: true,
							Path:   "path",
						}
						So(db.Insert(send).Run(), ShouldBeNil)

						outPath := gwRoot
						testFunc(send.ID, outPath)
					})
				})
			})
		})
	})
}
