package pipeline

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

func TestNewFileStream(t *testing.T) {
	logger := log.NewLogger("test_new_transfer_stream")

	Convey("Given a new database", t, func(c C) {
		ctx := initTestDB(c)

		trans := &model.Transfer{
			IsServer:   false,
			AgentID:    ctx.partner.ID,
			AccountID:  ctx.remoteAccount.ID,
			LocalPath:  filepath.Join(ctx.root, "file"),
			RemotePath: "/remote/file",
		}

		transCtx := &model.TransferContext{
			Transfer:      trans,
			RemoteAgent:   ctx.partner,
			RemoteAccount: ctx.remoteAccount,
			Paths:         &ctx.db.Conf.Paths,
		}

		Convey("Given a new send transfer", func(c C) {
			trans.RuleID = ctx.send.ID
			So(ctx.db.Insert(trans).Run(), ShouldBeNil)
			transCtx.Rule = ctx.send

			So(ioutil.WriteFile(trans.LocalPath, []byte("Hello World"), 0o700), ShouldBeNil)
			pip, err := newPipeline(ctx.db, logger, transCtx)
			So(err, ShouldBeNil)

			So(pip.machine.Transition("pre-tasks"), ShouldBeNil)
			So(pip.machine.Transition("pre-tasks done"), ShouldBeNil)
			So(pip.machine.Transition("start data"), ShouldBeNil)

			Convey("When creating a new transfer stream", func(c C) {
				stream, err := newFileStream(pip, time.Hour, false)
				So(err, ShouldBeNil)
				Reset(func() { _ = stream.file.Close() })

				Convey("Then it should  return a new transfer stream", func(c C) {
					So(stream, ShouldNotBeNil)

					Convey("Then the transfer file should have been opened", func(c C) {
						So(stream.file, ShouldNotBeNil)
						So(stream.file.Name(), ShouldEqual, trans.LocalPath)
					})
				})
			})

			Convey("Given that the file does not exist", func(c C) {
				So(os.Remove(trans.LocalPath), ShouldBeNil)

				Convey("When creating a new transfer stream", func(c C) {
					_, err := newFileStream(pip, time.Hour, false)

					Convey("Then it should return an error", func(c C) {
						So(err, ShouldBeError, types.NewTransferError(
							types.TeFileNotFound, "file not found"))
					})
				})
			})
		})

		Convey("Given a new receive transfer", func(c C) {
			trans.RuleID = ctx.recv.ID
			So(ctx.db.Insert(trans).Run(), ShouldBeNil)
			transCtx.Rule = ctx.recv
			pip, err := newPipeline(ctx.db, logger, transCtx)
			So(err, ShouldBeNil)

			So(pip.machine.Transition("pre-tasks"), ShouldBeNil)
			So(pip.machine.Transition("pre-tasks done"), ShouldBeNil)
			So(pip.machine.Transition("start data"), ShouldBeNil)

			Convey("When creating a new transfer stream", func(c C) {
				stream, err := newFileStream(pip, time.Hour, false)
				So(err, ShouldBeNil)
				Reset(func() { _ = stream.file.Close() })

				Convey("Then it should  return a new transfer stream", func(c C) {
					So(stream, ShouldNotBeNil)

					Convey("Then the transfer file should have been opened", func(c C) {
						So(stream.file, ShouldNotBeNil)
						So(stream.file.Name(), ShouldEqual, trans.LocalPath)
					})
				})
			})
		})
	})
}

func TestStreamRead(t *testing.T) {
	logger := log.NewLogger("test_stream_read")

	Convey("Given an outgoing transfer", t, func(c C) {
		ctx := initTestDB(c)

		trans := &model.Transfer{
			IsServer:   false,
			AgentID:    ctx.partner.ID,
			AccountID:  ctx.remoteAccount.ID,
			LocalPath:  filepath.Join(ctx.root, "file"),
			RemotePath: "/remote/file",
			RuleID:     ctx.send.ID,
		}
		So(ctx.db.Insert(trans).Run(), ShouldBeNil)

		transCtx := &model.TransferContext{
			Transfer:      trans,
			Rule:          ctx.send,
			RemoteAgent:   ctx.partner,
			RemoteAccount: ctx.remoteAccount,
			Paths:         &ctx.db.Conf.Paths,
		}

		Convey("Given a file stream for this transfer", func(c C) {
			content := []byte("read file content")
			So(ioutil.WriteFile(trans.LocalPath, content, 0600), ShouldBeNil)
			stream := initFilestream(ctx, logger, transCtx)

			Convey("When reading from the stream", func(c C) {
				b := make([]byte, 4)
				n, err := stream.Read(b)
				So(err, ShouldBeNil)

				Convey("Then it should return the correct number of bytes", func(c C) {
					So(n, ShouldEqual, len(b))
				})

				Convey("Then the transfer progression should have been updated", func(c C) {
					So(stream.progress, ShouldEqual, len(b))
				})

				Convey("Then the array should contain the file content", func(c C) {
					So(string(b), ShouldEqual, string(content[:len(b)]))
				})
			})

			Convey("Given that an error occurs while reading the file", func(c C) {
				_ = stream.file.Close()

				b := make([]byte, 4)
				_, err := stream.Read(b)
				So(err, ShouldBeError, "TransferError(TeDataTransfer): failed to read data")

				Convey("Then it should have called the error-tasks", func(c C) {
					waitEndTransfer(stream.Pipeline)

					Convey("Then any subsequent call to 'Read' should return an error", func(c C) {
						_, err := stream.Read(b)
						So(err, ShouldBeError, "TransferError(TeInternal): internal transfer error")
					})
				})
			})

			Convey("Given that database error occurs", func(c C) {
				database.SimulateError(c, ctx.db)
				time.Sleep(time.Microsecond) // just to be sure the ticker had the time to tick at least once

				b := make([]byte, 4)
				_, err := stream.Read(b)
				So(err, ShouldBeError, "TransferError(TeInternal): database error")

				Convey("Then it should have called the error-tasks", func(c C) {
					waitEndTransfer(stream.Pipeline)

					Convey("Then any subsequent call to 'Read' should return an error", func(c C) {
						_, err := stream.Read(b)
						So(err, ShouldBeError, "TransferError(TeInternal): internal transfer error")
					})
				})
			})
		})
	})
}

func TestStreamReadAt(t *testing.T) {
	logger := log.NewLogger("test_stream_read_at")

	Convey("Given an outgoing transfer", t, func(c C) {
		ctx := initTestDB(c)

		trans := &model.Transfer{
			IsServer:   false,
			AgentID:    ctx.partner.ID,
			AccountID:  ctx.remoteAccount.ID,
			LocalPath:  filepath.Join(ctx.root, "file"),
			RemotePath: "/remote/file",
			RuleID:     ctx.send.ID,
		}
		So(ctx.db.Insert(trans).Run(), ShouldBeNil)

		transCtx := &model.TransferContext{
			Transfer:      trans,
			Rule:          ctx.send,
			RemoteAgent:   ctx.partner,
			RemoteAccount: ctx.remoteAccount,
			Paths:         &ctx.db.Conf.Paths,
		}

		Convey("Given a file stream for this transfer", func(c C) {
			content := []byte("read file content")
			So(ioutil.WriteFile(trans.LocalPath, content, 0600), ShouldBeNil)
			stream := initFilestream(ctx, logger, transCtx)

			Convey("When reading from the stream with an offset", func(c C) {
				b := make([]byte, 4)
				off := 2
				n, err := stream.ReadAt(b, int64(off))
				So(err, ShouldBeNil)

				Convey("Then it should return the correct number of bytes", func(c C) {
					So(n, ShouldEqual, len(b))
				})

				Convey("Then the transfer progression should have been updated", func(c C) {
					So(stream.progress, ShouldEqual, len(b))
				})

				Convey("Then the array should contain the file content starting from the offset", func(c C) {
					So(string(b), ShouldEqual, string(content[off:off+len(b)]))
				})
			})

			Convey("Given that an error occurs while reading the file", func(c C) {
				_ = stream.file.Close()

				b := make([]byte, 4)
				_, err := stream.ReadAt(b, 0)
				So(err, ShouldBeError, "TransferError(TeDataTransfer): failed to read data")

				Convey("Then it should have called the error-tasks", func(c C) {
					waitEndTransfer(stream.Pipeline)

					Convey("Then any subsequent call to 'ReadAt' should return an error", func(c C) {
						_, err := stream.ReadAt(b, 0)
						So(err, ShouldBeError, "TransferError(TeInternal): internal transfer error")
					})
				})
			})

			Convey("Given that an error occurs while updating the progress", func(c C) {
				database.SimulateError(c, ctx.db)
				time.Sleep(time.Microsecond) // just to be sure the ticker had the time to tick

				b := make([]byte, 4)
				_, err := stream.ReadAt(b, 0)
				So(err, ShouldBeError, "TransferError(TeInternal): database error")

				Convey("Then it should have called the error-tasks", func(c C) {
					waitEndTransfer(stream.Pipeline)

					Convey("Then any subsequent call to 'ReadAt' should return an error", func(c C) {
						_, err := stream.ReadAt(b, 0)
						So(err, ShouldBeError, "TransferError(TeInternal): internal transfer error")
					})
				})
			})
		})
	})
}

func TestStreamWrite(t *testing.T) {
	logger := log.NewLogger("test_stream_write")

	Convey("Given an incoming transfer", t, func(c C) {
		ctx := initTestDB(c)

		trans := &model.Transfer{
			IsServer:   false,
			AgentID:    ctx.partner.ID,
			AccountID:  ctx.remoteAccount.ID,
			LocalPath:  filepath.Join(ctx.root, "file"),
			RemotePath: "/remote/file",
			RuleID:     ctx.recv.ID,
		}
		So(ctx.db.Insert(trans).Run(), ShouldBeNil)

		transCtx := &model.TransferContext{
			Transfer:      trans,
			Rule:          ctx.recv,
			RemoteAgent:   ctx.partner,
			RemoteAccount: ctx.remoteAccount,
			Paths:         &ctx.db.Conf.Paths,
		}

		Convey("Given a file stream for this transfer", func(c C) {
			stream := initFilestream(ctx, logger, transCtx)

			Convey("When writing to the stream", func(c C) {
				b := []byte("file content")
				n, err := stream.Write(b)
				So(err, ShouldBeNil)

				Convey("Then it should return the correct number of bytes", func(c C) {
					So(n, ShouldEqual, len(b))
				})

				Convey("Then the transfer progression should have been updated", func(c C) {
					So(stream.progress, ShouldEqual, len(b))
				})

				Convey("Then the file should contain the array content", func(c C) {
					content, err := ioutil.ReadFile(trans.LocalPath)
					So(err, ShouldBeNil)

					So(string(content), ShouldEqual, string(b))
				})
			})

			Convey("Given that an error occurs while writing the file", func(c C) {
				_ = stream.file.Close()

				b := make([]byte, 4)
				_, err := stream.Write(b)
				So(err, ShouldBeError, "TransferError(TeDataTransfer): failed to write data")

				Convey("Then it should have called the error-tasks", func(c C) {
					waitEndTransfer(stream.Pipeline)

					Convey("Then any subsequent call to 'Write' should return an error", func(c C) {
						_, err := stream.Write(b)
						So(err, ShouldBeError, "TransferError(TeInternal): internal transfer error")
					})
				})
			})

			Convey("Given that an error occurs while updating the progress", func(c C) {
				database.SimulateError(c, ctx.db)
				time.Sleep(time.Microsecond) // just to be sure the ticker had the time to tick

				b := make([]byte, 4)
				_, err := stream.Write(b)
				So(err, ShouldBeError, "TransferError(TeInternal): database error")

				Convey("Then it should have called the error-tasks", func(c C) {
					waitEndTransfer(stream.Pipeline)

					Convey("Then any subsequent call to 'Write' should return an error", func(c C) {
						_, err := stream.Write(b)
						So(err, ShouldBeError, "TransferError(TeInternal): internal transfer error")
					})
				})
			})
		})
	})
}

func TestStreamWriteAt(t *testing.T) {
	logger := log.NewLogger("test_stream_write_at")

	Convey("Given an incoming transfer", t, func(c C) {
		ctx := initTestDB(c)

		trans := &model.Transfer{
			IsServer:   false,
			AgentID:    ctx.partner.ID,
			AccountID:  ctx.remoteAccount.ID,
			LocalPath:  filepath.Join(ctx.root, "file"),
			RemotePath: "/remote/file",
			RuleID:     ctx.recv.ID,
		}
		So(ctx.db.Insert(trans).Run(), ShouldBeNil)

		transCtx := &model.TransferContext{
			Transfer:      trans,
			Rule:          ctx.recv,
			RemoteAgent:   ctx.partner,
			RemoteAccount: ctx.remoteAccount,
			Paths:         &ctx.db.Conf.Paths,
		}

		Convey("Given a file stream for this transfer", func(c C) {
			stream := initFilestream(ctx, logger, transCtx)

			Convey("When writing to the stream with an offset", func(c C) {
				b := []byte("file content")
				off := 2
				n, err := stream.WriteAt(b, int64(off))
				So(err, ShouldBeNil)

				Convey("Then it should return the correct number of bytes", func(c C) {
					So(n, ShouldEqual, len(b))
				})

				Convey("Then the transfer progression should have been updated", func(c C) {
					So(stream.progress, ShouldEqual, len(b))
				})

				Convey("Then the file should contain the array content with an offset", func(c C) {
					content, err := ioutil.ReadFile(trans.LocalPath)
					So(err, ShouldBeNil)

					So(string(content), ShouldEqual, strings.Repeat("\000", off)+string(b))
				})
			})

			Convey("Given that an error occurs while writing the file", func(c C) {
				_ = stream.file.Close()

				b := make([]byte, 4)
				_, err := stream.WriteAt(b, 0)
				So(err, ShouldBeError, "TransferError(TeDataTransfer): failed to write data")

				Convey("Then it should have called the error-tasks", func(c C) {
					waitEndTransfer(stream.Pipeline)

					Convey("Then any subsequent call to 'WriteAt' should return an error", func(c C) {
						_, err := stream.WriteAt(b, 0)
						So(err, ShouldBeError, "TransferError(TeInternal): internal transfer error")
					})
				})
			})

			Convey("Given that an error occurs while updating the progress", func(c C) {
				database.SimulateError(c, ctx.db)
				time.Sleep(time.Microsecond) // just to be sure the ticker had the time to tick

				b := make([]byte, 4)
				_, err := stream.WriteAt(b, 0)
				So(err, ShouldBeError, "TransferError(TeInternal): database error")

				Convey("Then it should have called the error-tasks", func(c C) {
					waitEndTransfer(stream.Pipeline)

					Convey("Then any subsequent call to 'WriteAt' should return an error", func(c C) {
						_, err := stream.WriteAt(b, 0)
						So(err, ShouldBeError, "TransferError(TeInternal): internal transfer error")
					})
				})
			})
		})
	})
}

func TestStreamClose(t *testing.T) {
	logger := log.NewLogger("test_stream_close")

	Convey("Given an incoming transfer", t, func(c C) {
		ctx := initTestDB(c)

		trans := &model.Transfer{
			IsServer:   false,
			AgentID:    ctx.partner.ID,
			AccountID:  ctx.remoteAccount.ID,
			LocalPath:  filepath.Join(ctx.root, "file"),
			RemotePath: "/remote/file",
			RuleID:     ctx.recv.ID,
		}
		So(ctx.db.Insert(trans).Run(), ShouldBeNil)

		transCtx := &model.TransferContext{
			Transfer:      trans,
			Rule:          ctx.recv,
			RemoteAgent:   ctx.partner,
			RemoteAccount: ctx.remoteAccount,
			Paths:         &ctx.db.Conf.Paths,
		}

		Convey("Given a file stream for this transfer", func(c C) {
			stream := initFilestream(ctx, logger, transCtx)
			stream.progress = 10
			So(stream.machine.Transition("end data"), ShouldBeNil)

			Convey("When closing the stream", func(c C) {
				So(stream.close(), ShouldBeNil)

				Convey("Then the underlying file should be closed", func(c C) {
					So(stream.file.Close(), ShouldBeError, fmt.Sprintf(
						"close %s: file already closed", trans.LocalPath))
				})

				Convey("Then subsequent call to `close` should return an error", func(c C) {
					So(stream.close(), ShouldBeError, errStateMachine)
					waitEndTransfer(stream.Pipeline)
				})
			})

			Convey("Given that an error occurs while updating the progress", func(c C) {
				database.SimulateError(c, ctx.db)
				So(stream.close(), ShouldBeError, "TransferError(TeInternal): database error")

				Convey("Then it should have called the error-tasks", func(c C) {
					waitEndTransfer(stream.Pipeline)
				})
			})
		})
	})
}

func TestStreamMove(t *testing.T) {
	logger := log.NewLogger("test_stream_move")

	Convey("Given an incoming transfer", t, func(c C) {
		ctx := initTestDB(c)

		trans := &model.Transfer{
			IsServer:   false,
			AgentID:    ctx.partner.ID,
			AccountID:  ctx.remoteAccount.ID,
			LocalPath:  filepath.Join(ctx.root, ctx.db.Conf.Paths.DefaultTmpDir, "file"),
			RemotePath: "/remote/file",
			RuleID:     ctx.recv.ID,
		}
		So(ctx.db.Insert(trans).Run(), ShouldBeNil)

		transCtx := &model.TransferContext{
			Transfer:      trans,
			Rule:          ctx.recv,
			RemoteAgent:   ctx.partner,
			RemoteAccount: ctx.remoteAccount,
			Paths:         &ctx.db.Conf.Paths,
		}

		Convey("Given a closed file stream for this transfer", func(c C) {
			stream := initFilestream(ctx, logger, transCtx)
			So(stream.machine.Transition("end data"), ShouldBeNil)
			So(stream.close(), ShouldBeNil)

			Convey("When moving the file", func(c C) {
				So(stream.move(), ShouldBeNil)

				Convey("Then the underlying file should have been be moved", func(c C) {
					_, err := os.Stat(filepath.Join(ctx.root, ctx.db.Conf.Paths.DefaultInDir, "file"))
					So(err, ShouldBeNil)
				})

				Convey("Then subsequent call to `move` should return an error", func(c C) {
					So(stream.move(), ShouldBeError, errStateMachine)
					waitEndTransfer(stream.Pipeline)
				})
			})

			Convey("Given that the move fails", func(c C) {
				So(os.Remove(stream.TransCtx.Transfer.LocalPath), ShouldBeNil)

				Convey("When moving the file", func(c C) {
					So(stream.move(), ShouldBeError, "TransferError(TeFinalization): "+
						"failed to move temp file")

					Convey("Then it should have called the error tasks", func(c C) {
						waitEndTransfer(stream.Pipeline)
					})
				})
			})

			Convey("Given that a database error occurs", func(c C) {
				database.SimulateError(c, ctx.db)

				Convey("When moving the file", func(c C) {
					So(stream.move(), ShouldBeError, "TransferError(TeInternal): "+
						"database error")

					Convey("Then it should have called the error tasks", func(c C) {
						waitEndTransfer(stream.Pipeline)
					})
				})
			})
		})
	})

	Convey("Given an outgoing transfer", t, func(c C) {
		ctx := initTestDB(c)

		trans := &model.Transfer{
			IsServer:   false,
			AgentID:    ctx.partner.ID,
			AccountID:  ctx.remoteAccount.ID,
			LocalPath:  filepath.Join(ctx.root, ctx.db.Conf.Paths.DefaultOutDir, "file"),
			RemotePath: "/remote/file",
			RuleID:     ctx.send.ID,
		}
		So(ctx.db.Insert(trans).Run(), ShouldBeNil)

		transCtx := &model.TransferContext{
			Transfer:      trans,
			Rule:          ctx.send,
			RemoteAgent:   ctx.partner,
			RemoteAccount: ctx.remoteAccount,
			Paths:         &ctx.db.Conf.Paths,
		}

		path := filepath.Join(ctx.root, ctx.db.Conf.Paths.DefaultOutDir, "file")
		So(ioutil.WriteFile(path, []byte("file content"), 0o700), ShouldBeNil)
		Reset(func() { _ = os.Remove(path) })

		Convey("Given a closed file stream for this transfer", func(c C) {
			stream := initFilestream(ctx, logger, transCtx)
			So(stream.machine.Transition("end data"), ShouldBeNil)
			So(stream.close(), ShouldBeNil)

			Convey("When moving the file", func(c C) {
				So(stream.move(), ShouldBeNil)

				Convey("Then it should do nothing", func(c C) {
					So(stream.TransCtx.Transfer.LocalPath, ShouldEqual, path)
				})
			})
		})
	})
}
