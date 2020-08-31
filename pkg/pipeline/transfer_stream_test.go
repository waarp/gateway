package pipeline

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
	. "github.com/smartystreets/goconvey/convey"
)

func TestNewTransferStream(t *testing.T) {
	logger := log.NewLogger("test_new_transfer_stream")
	cd, err := os.Getwd()
	if err != nil {
		t.FailNow()
	}
	root := filepath.Join(cd, "new_stream_root")
	paths := Paths{PathsConfig: conf.PathsConfig{
		GatewayHome:   root,
		InDirectory:   filepath.Join(root, "in"),
		OutDirectory:  filepath.Join(root, "out"),
		WorkDirectory: filepath.Join(root, "work"),
	}}

	Convey("Given a new transfer", t, func() {
		TransferInCount = &Count{}
		TransferOutCount = &Count{}

		db := database.GetTestDatabase()

		rule := &model.Rule{
			Name:   "rule",
			IsSend: false,
			Path:   "rule/path",
		}
		So(db.Create(rule), ShouldBeNil)

		agent := &model.RemoteAgent{
			Name:        "agent",
			Protocol:    "test",
			ProtoConfig: []byte(`{}`),
		}
		So(db.Create(agent), ShouldBeNil)

		account := &model.RemoteAccount{
			RemoteAgentID: agent.ID,
			Login:         "login",
			Password:      []byte("password"),
		}
		So(db.Create(account), ShouldBeNil)

		trans := model.Transfer{
			RuleID:     rule.ID,
			IsServer:   false,
			AgentID:    agent.ID,
			AccountID:  account.ID,
			SourceFile: "source",
			DestFile:   "dest",
		}

		Convey("Given no transfer limit", func() {
			TransferInCount.SetLimit(0)
			Reset(func() { TransferInCount = &Count{} })

			Convey("When creating a new transfer stream", func() {
				stream, err := NewTransferStream(context.Background(), logger,
					db, paths, &trans)
				So(err, ShouldBeNil)

				Convey("Then it should  return a new transfer stream", func() {
					So(stream, ShouldNotBeNil)
				})
			})
		})

		Convey("Given a transfer limit", func() {
			TransferOutCount.SetLimit(1)
			So(TransferOutCount.add(), ShouldBeNil)
			Reset(func() { TransferOutCount = &Count{} })

			Convey("When creating a new transfer stream", func() {
				_, err := NewTransferStream(context.Background(), logger, db,
					paths, &trans)

				Convey("Then it should return an error", func() {
					So(err, ShouldBeError, ErrLimitReached)
				})
			})
		})
	})
}

func TestStreamRead(t *testing.T) {
	logger := log.NewLogger("test_stream_read")

	cd, err := os.Getwd()
	if err != nil {
		t.FailNow()
	}
	root := filepath.Join(cd, "stream_read_root")
	paths := Paths{PathsConfig: conf.PathsConfig{
		GatewayHome:   root,
		InDirectory:   filepath.Join(root, "in"),
		OutDirectory:  filepath.Join(root, ""),
		WorkDirectory: filepath.Join(root, "work"),
	}}

	Convey("Given a file", t, func() {
		srcFile := "read_test.src"
		dstFile := "read_test.dst"
		So(os.Mkdir(root, 0700), ShouldBeNil)
		path := filepath.Join(root, srcFile)
		content := []byte("Transfer stream read test content")
		So(ioutil.WriteFile(path, content, 0600), ShouldBeNil)

		Reset(func() { _ = os.RemoveAll(root) })

		Convey("Given a transfer stream to this file", func() {
			db := database.GetTestDatabase()
			rule := &model.Rule{
				Name:    "rule",
				Comment: "",
				IsSend:  true,
				Path:    "path",
				InPath:  ".",
				OutPath: ".",
			}
			So(db.Create(rule), ShouldBeNil)

			agent := &model.LocalAgent{
				Owner:       database.Owner,
				Name:        "agent",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(agent), ShouldBeNil)

			account := &model.LocalAccount{
				LocalAgentID: agent.ID,
				Login:        "login",
				Password:     []byte("password"),
			}
			So(db.Create(account), ShouldBeNil)

			So(err, ShouldBeNil)
			trans := &model.Transfer{
				RuleID:     rule.ID,
				IsServer:   true,
				AgentID:    agent.ID,
				AccountID:  account.ID,
				SourceFile: srcFile,
				DestFile:   dstFile,
				Start:      time.Now(),
				Status:     model.StatusRunning,
				Owner:      database.Owner,
				Progress:   0,
				TaskNumber: 0,
				Error:      model.TransferError{},
			}
			So(db.Create(trans), ShouldBeNil)

			stream, tErr := NewTransferStream(context.Background(), logger, db, paths, trans)
			So(tErr, ShouldBeNil)
			Reset(func() { _ = stream.Close() })

			So(stream.Start(), ShouldBeNil)

			Convey("When reading the stream", func() {
				b := make([]byte, 4)

				n, err := stream.Read(b)

				Convey("Then it should NOT return an error", func() {
					So(err, ShouldBeNil)

					Convey("Then it should return the correct number of bytes", func() {
						So(n, ShouldEqual, len(b))
					})

					Convey("Then the transfer progression should have been updated", func() {
						t := &model.Transfer{ID: trans.ID}
						So(db.Get(t), ShouldBeNil)

						So(t.Progress, ShouldEqual, len(b))
					})

					Convey("Then the array should contain the file content", func() {
						content, err := ioutil.ReadFile(path)
						So(err, ShouldBeNil)

						So(string(b), ShouldEqual, string(content[:len(b)]))
					})
				})
			})

			Convey("When reading the stream with an offset", func() {
				b := make([]byte, 4)

				off := 2
				n, err := stream.ReadAt(b, int64(off))

				Convey("Then it should NOT return an error", func() {
					So(err, ShouldBeNil)

					Convey("Then it should return the correct number of bytes", func() {
						So(n, ShouldEqual, len(b))
					})

					Convey("Then the transfer progression should have been updated", func() {
						t := &model.Transfer{ID: trans.ID}
						So(db.Get(t), ShouldBeNil)

						So(t.Progress, ShouldEqual, len(b))
					})

					Convey("Then the array should contain the file content", func() {
						content, err := ioutil.ReadFile(path)
						So(err, ShouldBeNil)

						So(string(b), ShouldEqual, string(content[off:off+len(b)]))
					})
				})
			})
		})
	})
}

func TestStreamWrite(t *testing.T) {
	logger := log.NewLogger("test_stream_read")

	cd, err := os.Getwd()
	if err != nil {
		t.FailNow()
	}
	root := filepath.Join(cd, "stream_write_root")
	paths := Paths{PathsConfig: conf.PathsConfig{
		GatewayHome:   root,
		InDirectory:   filepath.Join(root, "in"),
		OutDirectory:  filepath.Join(root, "out"),
		WorkDirectory: filepath.Join(root, "work"),
	}}

	Convey("Given a file", t, func() {
		dstFile := "write_test.dst"
		content := []byte("Transfer stream write test content")
		So(os.Mkdir(root, 0700), ShouldBeNil)
		Reset(func() { So(os.RemoveAll(root), ShouldBeNil) })

		Convey("Given a transfer stream", func() {
			db := database.GetTestDatabase()
			rule := &model.Rule{
				Name:   "rule",
				IsSend: false,
				Path:   ".",
			}
			So(db.Create(rule), ShouldBeNil)

			agent := &model.LocalAgent{
				Owner:       database.Owner,
				Name:        "agent",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(agent), ShouldBeNil)

			account := &model.LocalAccount{
				LocalAgentID: agent.ID,
				Login:        "login",
				Password:     []byte("password"),
			}
			So(db.Create(account), ShouldBeNil)

			trans := &model.Transfer{
				RuleID:     rule.ID,
				IsServer:   true,
				AgentID:    agent.ID,
				AccountID:  account.ID,
				SourceFile: "write_test.src",
				DestFile:   dstFile,
				Start:      time.Now(),
				Status:     model.StatusRunning,
				Owner:      database.Owner,
				Progress:   0,
				TaskNumber: 0,
				Error:      model.TransferError{},
			}
			So(db.Create(trans), ShouldBeNil)

			stream, tErr := NewTransferStream(context.Background(), logger, db, paths, trans)
			So(tErr, ShouldBeNil)
			Reset(func() { _ = stream.Close() })

			So(stream.Start(), ShouldBeNil)

			Convey("When writing to the stream", func() {
				w := content[:15]
				n, err := stream.Write(w)

				Convey("Then it should NOT return an error", func() {
					So(err, ShouldBeNil)

					Convey("Then it should return the correct number of bytes", func() {
						So(n, ShouldEqual, len(w))
					})

					Convey("Then the transfer progression should have been updated", func() {
						t := &model.Transfer{ID: trans.ID}
						So(db.Get(t), ShouldBeNil)

						So(t.Progress, ShouldEqual, len(w))
					})

					Convey("Then the file should contain the array content", func() {
						s, err := ioutil.ReadFile(utils.DenormalizePath(stream.Transfer.TrueFilepath))
						So(err, ShouldBeNil)

						So(string(s), ShouldEqual, string(w))
					})
				})
			})

			Convey("When writing to the stream with an offset", func() {
				w := content[:15]

				off := 5
				n, err := stream.WriteAt(w, int64(off))

				Convey("Then it should NOT return an error", func() {
					So(err, ShouldBeNil)

					Convey("Then it should return the correct number of bytes", func() {
						So(n, ShouldEqual, len(w))
					})

					Convey("Then the transfer progression should have been updated", func() {
						t := &model.Transfer{ID: trans.ID}
						So(db.Get(t), ShouldBeNil)

						So(t.Progress, ShouldEqual, len(w))
					})

					Convey("Then the file should contain the array content", func() {
						s, err := ioutil.ReadFile(utils.DenormalizePath(stream.Transfer.TrueFilepath))
						So(err, ShouldBeNil)

						So(string(s[off:]), ShouldEqual, string(w))
					})
				})
			})
		})
	})
}
