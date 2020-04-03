package pipeline

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	. "github.com/smartystreets/goconvey/convey"
)

func TestNewTransferStream(t *testing.T) {
	logger := log.NewLogger("test_new_transfer_stream")

	Convey("Given a new transfer", t, func() {
		TransferInCount = &Count{}
		TransferOutCount = &Count{}

		db := database.GetTestDatabase()

		rule := &model.Rule{
			Name:   "rule",
			IsSend: false,
			Path:   ".",
		}
		So(db.Create(rule), ShouldBeNil)

		trans := model.Transfer{
			ID:         1,
			RuleID:     1,
			IsServer:   true,
			AgentID:    1,
			AccountID:  1,
			SourcePath: ".",
			DestPath:   ".",
		}

		Convey("Given no transfer limit", func() {
			TransferInCount.SetLimit(0)
			Reset(func() { TransferInCount = &Count{} })

			Convey("When creating a new transfer stream", func() {
				stream, err := NewTransferStream(context.Background(), logger,
					db, ".", trans)
				So(err, ShouldBeNil)

				Convey("Then it should  return a new transfer stream", func() {
					So(stream, ShouldNotBeNil)
				})
			})
		})

		Convey("Given a transfer limit", func() {
			TransferInCount.SetLimit(1)
			So(TransferInCount.add(), ShouldBeNil)
			Reset(func() { TransferInCount = &Count{} })

			Convey("When creating a new transfer stream", func() {
				_, err := NewTransferStream(context.Background(), logger, db,
					".", trans)

				Convey("Then it should return an error", func() {
					So(err, ShouldBeError, ErrLimitReached)
				})
			})
		})
	})
}

func TestStreamRead(t *testing.T) {
	logger := log.NewLogger("test_stream_read")

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

		stream, tErr := NewTransferStream(context.Background(), logger, db, ".", *trans)
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
	logger := log.NewLogger("test_stream_read")

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

		stream, tErr := NewTransferStream(context.Background(), logger, db, ".", *trans)
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
