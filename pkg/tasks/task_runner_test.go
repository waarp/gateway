package tasks

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/logtest"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func TestSetup(t *testing.T) {
	t.Parallel()

	root := filepath.ToSlash(t.TempDir())
	rootAlt := filepath.ToSlash(t.TempDir())

	Convey("Given a Task with some replacement variables", t, func(c C) {
		const (
			timestampTsFormat  = "YYYY-MM-DD_HH:mm:sszz"
			timestampGoFormat  = "2006-01-02_15:04:05Z07:00"
			timestampPrecision = time.Second
			defaultTsGoFormat  = "2006-01-02_150405"
		)

		task := &model.Task{
			Type: "DUMMY",
			Args: map[string]string{
				"rule": "#RULE#", "date": "#DATE#", "hour": "#HOUR#",
				"trueFullPath": "#TRUEFULLPATH#", "trueFilename": "#TRUEFILENAME#",
				"fullPath": "#ORIGINALFULLPATH#", "filename": "#ORIGINALFILENAME#",
				"remoteHost": "#REMOTEHOST#", "localHost": "#LOCALHOST#",
				"transferID": "#TRANSFERID#", "requesterHost": "#REQUESTERHOST#",
				"requestedHost": "#REQUESTEDHOST#", "fullTransferID": "#FULLTRANSFERID#",
				"errCode": "#ERRORCODE#", "errMsg": "#ERRORMSG#", "errStrCode": "#ERRORSTRCODE#",
				"inPath": "#INPATH#", "outPath": "#OUTPATH#", "workPath": "#WORKPATH#",
				"homePath": "#HOMEPATH#", "transferInfo": "#TI_foo#/#TI_id#",
				"baseFileName": "#BASEFILENAME#", "fileExtension": "#FILEEXTENSION#",
				"timestamp":             fmt.Sprintf("#TIMESTAMP(%s)#", timestampTsFormat),
				"defaultTimestamp":      `#TIMESTAMP#`,
				"startTimestamp":        fmt.Sprintf(`#STARTTIMESTAMP(%s)#`, timestampTsFormat),
				"defaultStartTimestamp": "#STARTTIMESTAMP#",
				"remoteTransferID":      "#REMOTETRANSFERID#",
			},
		}

		Convey("Given a Runner", func(c C) {
			// The database is set to nil here because the test does not require
			// it. TestDatabase is only used for its side effect of initializing
			// the configiration. It speds up the test by a 150x factor.

			// db := database.TestDatabase(c)
			var db *database.DB

			const (
				agentID    = 1
				accountID  = 10
				clientID   = 100
				ruleID     = 1000
				transferID = 1234
			)

			transCtx := &model.TransferContext{
				Paths: &conf.PathsConfig{
					GatewayHome:   root,
					DefaultInDir:  "in_dir",
					DefaultOutDir: "out_dir",
					DefaultTmpDir: path.Join(rootAlt, "tmp_dir"),
				},
				Client: &model.Client{
					Identifier: model.ID(clientID),
					Name:       "cli",
					Protocol:   testProtocol,
				},
				RemoteAgent: &model.RemoteAgent{
					Identifier: model.ID(agentID),
					Name:       "partner",
					Protocol:   testProtocol,
					Address:    types.Addr("localhost", 6622),
				},
				RemoteAccount: &model.RemoteAccount{
					Identifier:    model.ID(accountID),
					RemoteAgentID: agentID,
					Login:         "toto",
				},
				Rule: &model.Rule{
					Identifier:     model.ID(ruleID),
					Name:           "rulename",
					IsSend:         true,
					Path:           "path/to/test",
					LocalDir:       "local/dir",
					RemoteDir:      "remote/dir",
					TmpLocalRcvDir: "local/tmp",
				},
			}

			transCtx.Transfer = &model.Transfer{
				Identifier:       model.ID(transferID),
				RemoteTransferID: "123456",
				RemoteAccountID:  transCtx.RemoteAccount.NullableID(),
				ClientID:         transCtx.Client.NullableID(),
				SrcFilename:      "src/file",
				DestFilename:     "dst/file",
				LocalPath:        path.Join(root, transCtx.Rule.LocalDir, "file.test"),
				RemotePath:       path.Join(transCtx.Rule.RemoteDir, "file.rem"),
				ErrCode:          types.TeConnection,
				ErrDetails:       `error message`,
				TransferInfo:     map[string]any{"foo": "bar", "id": 123},
				Start:            time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			}

			r := &Runner{db: db, transCtx: transCtx}

			Convey("When calling the `setup` function", func() {
				now := time.Now()
				res, err := r.setup(task)

				Convey("Then it should NOT return an error", func() {
					So(err, ShouldBeNil)

					Convey("Then res should contain an entry `rule`", func() {
						val, ok := res["rule"]
						So(ok, ShouldBeTrue)

						Convey("Then res[rule] should contain the resolved variable", func() {
							So(val, ShouldEqual, r.transCtx.Rule.Name)
						})
					})

					Convey("Then res should contain an entry `date`", func() {
						val, ok := res["date"]
						So(ok, ShouldBeTrue)

						Convey("Then res[date] should contain the resolved variable", func() {
							So(val, ShouldEqual, now.Format("20060102"))
						})
					})

					Convey("Then res should contain an entry `hour`", func() {
						val, ok := res["hour"]
						So(ok, ShouldBeTrue)

						Convey("Then res[hour] should contain the resolved variable", func() {
							So(val, ShouldEqual, now.Format("150405"))
						})
					})

					Convey("Then res should contain an entry `trueFullPath`", func() {
						val, ok := res["trueFullPath"]
						So(ok, ShouldBeTrue)

						Convey("Then res[trueFullPath] should contain the resolved variable", func() {
							So(val, ShouldEqual, r.transCtx.Transfer.LocalPath)
						})
					})

					Convey("Then res should contain an entry `trueFilename`", func() {
						val, ok := res["trueFilename"]
						So(ok, ShouldBeTrue)

						Convey("Then res[trueFilename] should contain the resolved variable", func() {
							So(val, ShouldEqual, path.Base(r.transCtx.Transfer.LocalPath))
						})
					})

					Convey("Then res should contain an entry `fullPath`", func() {
						val, ok := res["fullPath"]
						So(ok, ShouldBeTrue)

						Convey("Then res[fullPath] should contain the resolved variable", func() {
							So(val, ShouldEqual, r.transCtx.Transfer.LocalPath)
						})
					})

					Convey("Then res should contain an entry `filename`", func() {
						val, ok := res["filename"]
						So(ok, ShouldBeTrue)

						Convey("Then res[filename] should contain the resolved variable", func() {
							So(val, ShouldEqual, path.Base(transCtx.Transfer.SrcFilename))
						})
					})

					Convey("Then res should contain an entry `remoteHost`", func() {
						val, ok := res["remoteHost"]
						So(ok, ShouldBeTrue)

						Convey("Then res[remoteHost] should contain the resolved variable", func() {
							So(val, ShouldEqual, transCtx.RemoteAgent.Name)
						})
					})

					Convey("Then res should contain an entry `localHost`", func() {
						val, ok := res["localHost"]
						So(ok, ShouldBeTrue)

						Convey("Then res[localHost] should contain the resolved variable", func() {
							So(val, ShouldEqual, transCtx.RemoteAccount.Login)
						})
					})

					Convey("Then res should contain an entry `transferID`", func() {
						val, ok := res["transferID"]
						So(ok, ShouldBeTrue)

						Convey("Then res[transferID] should contain the resolved variable", func() {
							So(val, ShouldEqual, utils.FormatInt(r.transCtx.Transfer.ID))
						})
					})

					Convey("Then res should contain an entry `remoteRransferID`", func() {
						val, ok := res["remoteTransferID"]
						So(ok, ShouldBeTrue)

						Convey("Then res[remoteTransferID] should contain the resolved variable", func() {
							So(val, ShouldEqual, r.transCtx.Transfer.RemoteTransferID)
						})
					})

					Convey("Then res should contain an entry `requesterHost`", func() {
						val, ok := res["requesterHost"]
						So(ok, ShouldBeTrue)

						Convey("Then res[requesterHost] should contain the resolved variable", func() {
							So(val, ShouldEqual, transCtx.RemoteAccount.Login)
						})
					})

					Convey("Then res should contain an entry `requestedHost`", func() {
						val, ok := res["requestedHost"]
						So(ok, ShouldBeTrue)

						Convey("Then res[requestedHost] should contain the resolved variable", func() {
							So(val, ShouldEqual, transCtx.RemoteAgent.Name)
						})
					})

					Convey("Then res should contain an entry `fullTransferID`", func() {
						val, ok := res["fullTransferID"]
						So(ok, ShouldBeTrue)

						Convey("Then res[fullTransferID] should contain the resolved variable", func() {
							So(val, ShouldEqual, fmt.Sprintf("%d_%s_%s", transferID,
								transCtx.RemoteAccount.Login, transCtx.RemoteAgent.Name))
						})
					})

					Convey("Then res should contain an entry `errCode`", func() {
						val, ok := res["errCode"]
						So(ok, ShouldBeTrue)

						Convey("Then res[errCode] should contain the resolved variable", func() {
							So(val, ShouldEqual, string(r.transCtx.Transfer.ErrCode.R66Code()))
						})
					})

					Convey("Then res should contain an entry `errMsg`", func() {
						val, ok := res["errMsg"]
						So(ok, ShouldBeTrue)

						Convey("Then res[msg] should contain the resolved variable", func() {
							So(val, ShouldEqual, r.transCtx.Transfer.ErrDetails)
						})
					})

					Convey("Then res should contain an entry `errStrCode`", func() {
						val, ok := res["errStrCode"]
						So(ok, ShouldBeTrue)

						Convey("Then res[errStrCode] should contain the resolved variable", func() {
							So(val, ShouldEqual, r.transCtx.Transfer.ErrDetails)
						})
					})

					Convey("Then res should contain an entry `homePath`", func() {
						val, ok := res["homePath"]
						So(ok, ShouldBeTrue)

						Convey("Then res[homePath] should contain the resolved variable", func() {
							So(val, ShouldEqual, root)
						})
					})

					Convey("Then res should contain an entry `inPath`", func() {
						val, ok := res["inPath"]
						So(ok, ShouldBeTrue)

						Convey("Then res[inPath] should contain the resolved variable", func() {
							So(val, ShouldEqual, fs.JoinPath(root, r.transCtx.Paths.DefaultInDir))
						})
					})

					Convey("Then res should contain an entry `outPath`", func() {
						val, ok := res["outPath"]
						So(ok, ShouldBeTrue)

						Convey("Then res[outPath] should contain the resolved variable", func() {
							So(val, ShouldEqual, fs.JoinPath(root, r.transCtx.Rule.LocalDir))
						})
					})

					Convey("Then res should contain an entry `workPath`", func() {
						val, ok := res["workPath"]
						So(ok, ShouldBeTrue)

						Convey("Then res[workPath] should contain the resolved variable", func() {
							So(val, ShouldEqual, r.transCtx.Paths.DefaultTmpDir)
						})
					})

					Convey("Then res should contain a `transferInfo` entry", func() {
						val, ok := res["transferInfo"]
						So(ok, ShouldBeTrue)

						Convey("Then res[transferInfo] should contain the resolved variable", func() {
							So(val, ShouldEqual, "bar/123")
						})
					})

					Convey("Then res should contain a `baseFileName` entry", func() {
						val, ok := res["baseFileName"]
						So(ok, ShouldBeTrue)

						Convey("Then res[transferInfo] should contain the resolved variable", func() {
							So(val, ShouldEqual, "file")
						})
					})

					Convey("Then res should contain a `fileExtension` entry", func() {
						val, ok := res["fileExtension"]
						So(ok, ShouldBeTrue)

						Convey("Then res[transferInfo] should contain the resolved variable", func() {
							So(val, ShouldEqual, ".test")
						})
					})

					Convey("Then res should contain a `timestamp` entry", func() {
						val, ok := res["timestamp"]
						So(ok, ShouldBeTrue)

						Convey("Then res[transferInfo] should contain the resolved variable", func() {
							valT, tErr := time.ParseInLocation(timestampGoFormat, val, time.Local)
							So(tErr, ShouldBeNil)
							So(valT, ShouldHappenWithin, timestampPrecision, now)
						})
					})

					Convey("Then res should contain a `defaultTimestamp` entry", func() {
						val, ok := res["defaultTimestamp"]
						So(ok, ShouldBeTrue)

						Convey("Then res[transferInfo] should contain the resolved variable", func() {
							valT, tErr := time.ParseInLocation(defaultTsGoFormat, val, time.Local)
							So(tErr, ShouldBeNil)
							So(valT, ShouldHappenWithin, timestampPrecision, now)
						})
					})

					Convey("Then res should contain a `startTimestamp` entry", func() {
						val, ok := res["startTimestamp"]
						So(ok, ShouldBeTrue)

						Convey("Then res[transferInfo] should contain the resolved variable", func() {
							valT, tErr := time.ParseInLocation(timestampGoFormat, val, time.Local)
							So(tErr, ShouldBeNil)
							So(valT, ShouldHappenWithin, time.Duration(0), transCtx.Transfer.Start)
						})
					})

					Convey("Then res should contain a `defaultStartTimestamp` entry", func() {
						val, ok := res["defaultStartTimestamp"]
						So(ok, ShouldBeTrue)

						Convey("Then res[transferInfo] should contain the resolved variable", func() {
							valT, tErr := time.ParseInLocation(defaultTsGoFormat, val, time.Local)
							So(tErr, ShouldBeNil)
							So(valT, ShouldHappenWithin, time.Duration(0), transCtx.Transfer.Start)
						})
					})
				})
			})
		})
	})
}

func TestRunTasks(t *testing.T) {
	setup := func(tb testing.TB) *Runner {
		tb.Helper()
		logger := logtest.GetTestLogger(tb)
		db := dbtest.TestDatabase(tb)

		rule := &model.Rule{Name: "rule", IsSend: true, Path: "path"}
		require.NoError(tb, db.Insert(rule).Run())

		client := &model.Client{Name: "cli", Protocol: testProtocol}
		require.NoError(tb, db.Insert(client).Run())

		agent := &model.RemoteAgent{
			Name:     "agent",
			Protocol: client.Protocol,
			Address:  types.Addr("localhost", 6622),
		}
		require.NoError(tb, db.Insert(agent).Run())

		account := &model.RemoteAccount{
			RemoteAgentID: agent.ID,
			Login:         "toto",
		}
		require.NoError(tb, db.Insert(account).Run())

		trans := &model.Transfer{
			RuleID:          rule.ID,
			ClientID:        client.NullableID(),
			RemoteAccountID: account.NullableID(),
			SrcFilename:     "/src/file",
			DestFilename:    "/dst/file",
			LocalPath:       filepath.Join(tb.TempDir(), "file"),
		}
		require.NoError(tb, db.Insert(trans).Run())

		return &Runner{
			db:     db,
			Logger: logger,
			transCtx: &model.TransferContext{
				Rule:     rule,
				Transfer: trans,
			},
			Ctx: tb.Context(),
		}
	}

	t.Run("Success", func(t *testing.T) {
		proc := setup(t)
		dummyTaskCheck = make(chan string, 3)
		tasks := []*model.Task{
			{
				RuleID: proc.transCtx.Rule.ID,
				Chain:  model.ChainPre,
				Rank:   0,
				Type:   taskSuccess,
				Args:   map[string]string{},
			}, {
				RuleID: proc.transCtx.Rule.ID,
				Chain:  model.ChainPre,
				Rank:   1,
				Type:   taskSuccess,
				Args:   map[string]string{},
			},
		}

		rv := proc.runTasks(tasks, nil)
		require.Nil(t, rv)

		dummyTaskCheck <- "DONE"

		assert.Equal(t, "SUCCESS", <-dummyTaskCheck)
		assert.Equal(t, "SUCCESS", <-dummyTaskCheck)
		assert.Equal(t, "DONE", <-dummyTaskCheck)
	})

	t.Run("Warning", func(t *testing.T) {
		proc := setup(t)
		dummyTaskCheck = make(chan string, 3)
		tasks := []*model.Task{
			{
				RuleID: proc.transCtx.Rule.ID,
				Chain:  model.ChainPre,
				Rank:   0,
				Type:   taskSuccess,
				Args:   map[string]string{},
			}, {
				RuleID: proc.transCtx.Rule.ID,
				Chain:  model.ChainPre,
				Rank:   1,
				Type:   taskWarning,
				Args:   map[string]string{},
			},
		}

		rv := proc.runTasks(tasks, nil)
		require.Nil(t, rv)

		dummyTaskCheck <- "DONE"

		assert.Equal(t, "SUCCESS", <-dummyTaskCheck)
		assert.Equal(t, "WARNING", <-dummyTaskCheck)
		assert.Equal(t, "DONE", <-dummyTaskCheck)

		assert.Equal(t, types.TeWarning, proc.transCtx.Transfer.ErrCode)
		assert.Equal(t, "Task TESTWARNING @ rule PRE[1]: warning message",
			proc.transCtx.Transfer.ErrDetails)
	})

	t.Run("Failure", func(t *testing.T) {
		proc := setup(t)
		dummyTaskCheck = make(chan string, 3)
		tasks := []*model.Task{
			{
				RuleID: proc.transCtx.Rule.ID,
				Chain:  model.ChainPre,
				Rank:   0,
				Type:   taskSuccess,
				Args:   map[string]string{},
			}, {
				RuleID: proc.transCtx.Rule.ID,
				Chain:  model.ChainPre,
				Rank:   1,
				Type:   taskFail,
				Args:   map[string]string{},
			}, {
				RuleID: proc.transCtx.Rule.ID,
				Chain:  model.ChainPre,
				Rank:   1,
				Type:   taskSuccess,
				Args:   map[string]string{},
			},
		}

		rv := proc.runTasks(tasks, nil)
		require.Error(t, rv)
		dummyTaskCheck <- "DONE"

		assert.Equal(t, "SUCCESS", <-dummyTaskCheck)
		assert.Equal(t, "FAILURE", <-dummyTaskCheck)
		assert.Equal(t, "DONE", <-dummyTaskCheck)
	})

	t.Run("Unknown task", func(t *testing.T) {
		proc := setup(t)
		dummyTaskCheck = make(chan string, 1)
		tasks := []*model.Task{{
			RuleID: proc.transCtx.Rule.ID,
			Chain:  model.ChainPre,
			Rank:   0,
			Type:   "unknown",
			Args:   map[string]string{},
		}}

		assert.ErrorContains(t, proc.runTasks(tasks, nil), "unknown task type")
	})

	t.Run("Delete", func(t *testing.T) {
		proc := setup(t)
		logOut := &strings.Builder{}
		proc.Logger = logtest.GetTestLogger(t,
			logtest.WithLevel("WARNING"),
			logtest.WithWriter(logOut))

		require.NoError(t, os.WriteFile(proc.transCtx.Transfer.LocalPath, []byte{}, 0o600))

		tasks := []*model.Task{{
			RuleID: proc.transCtx.Rule.ID,
			Chain:  model.ChainPre,
			Rank:   0,
			Type:   Delete,
			Args:   map[string]string{},
		}}

		rv := proc.runTasks(tasks, nil)
		require.Nil(t, rv)

		assert.Empty(t, logOut.String())
	})
}
