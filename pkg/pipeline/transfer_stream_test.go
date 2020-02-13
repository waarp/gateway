package pipeline

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	. "github.com/smartystreets/goconvey/convey"
)

var logConf = conf.LogConfig{
	Level: "DEBUG",
	LogTo: "stdout",
}

func TestStreamRead(t *testing.T) {
	logger := log.NewLogger("test_stream_read", logConf)

	filename := "test_stream_read.src"
	content := "Transfer stream read test content"

	Convey("Given a transfer stream", t, func() {
		err := ioutil.WriteFile(filename, []byte(content), 0600)
		So(err, ShouldBeNil)
		Reset(func() { _ = os.Remove(filename) })

		db := database.GetTestDatabase()
		rule := &model.Rule{
			Name:    "rule",
			Comment: "",
			IsSend:  true,
			Path:    ".",
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
			SourcePath: filename,
			DestPath:   ".",
			Start:      time.Now(),
			Status:     model.StatusRunning,
			Owner:      database.Owner,
			Progress:   0,
			TaskNumber: 0,
			Error:      model.TransferError{},
		}
		So(db.Create(trans), ShouldBeNil)

		stream, tErr := NewTransferStream(logger, db, ".", *trans)
		So(tErr, ShouldBeNil)

		So(stream.Start(), ShouldBeNil)

		Convey("When reading the stream", func() {
			b := make([]byte, 15)

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
					So(string(b), ShouldEqual, content[:len(b)])
				})
			})
		})

		Convey("When reading the stream with an offset", func() {
			b := make([]byte, 15)

			off := 5
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
					So(string(b), ShouldEqual, content[off:off+len(b)])
				})
			})
		})
	})
}

func TestStreamWrite(t *testing.T) {
	logger := log.NewLogger("test_stream_read", logConf)

	filename := "test_stream_write.dst"
	file := filepath.Join("tmp", filename)
	content := "Transfer stream write test content"

	Convey("Given a transfer stream", t, func() {
		db := database.GetTestDatabase()
		rule := &model.Rule{
			Name:    "rule",
			Comment: "",
			IsSend:  false,
			Path:    ".",
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
			SourcePath: ".",
			DestPath:   filename,
			Start:      time.Now(),
			Status:     model.StatusRunning,
			Owner:      database.Owner,
			Progress:   0,
			TaskNumber: 0,
			Error:      model.TransferError{},
		}
		So(db.Create(trans), ShouldBeNil)

		stream, tErr := NewTransferStream(logger, db, ".", *trans)
		So(tErr, ShouldBeNil)

		So(stream.Start(), ShouldBeNil)
		Reset(func() { _ = os.RemoveAll("tmp") })

		Convey("When writing to the stream", func() {
			b := []byte(content[:15])

			n, err := stream.Write(b)

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

				Convey("Then the file should contain the array content", func() {
					s, err := ioutil.ReadFile(file)
					So(err, ShouldBeNil)

					So(string(s), ShouldEqual, string(b))
				})
			})
		})

		Convey("When writing to the stream with an offset", func() {
			b := []byte(content[:15])

			off := 5
			n, err := stream.WriteAt(b, int64(off))

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

				Convey("Then the file should contain the array content", func() {
					s, err := ioutil.ReadFile(file)
					So(err, ShouldBeNil)

					So(string(s[off:]), ShouldEqual, string(b))
				})
			})
		})
	})
}
