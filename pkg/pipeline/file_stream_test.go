package pipeline

import (
	"io"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var (
	errRead  = NewError(types.TeInternal, "failed to read file")
	errWrite = NewError(types.TeInternal, "failed to write file")
)

func TestNewFileStream(t *testing.T) {
	root := t.TempDir()

	Convey("Given a new database", t, func(c C) {
		ctx := initTestDB(c, root)

		trans := &model.Transfer{
			ClientID:        utils.NewNullInt64(ctx.client.ID),
			RemoteAccountID: utils.NewNullInt64(ctx.remoteAccount.ID),
		}

		Convey("Given a new send transfer", func(c C) {
			trans.RuleID = ctx.send.ID
			trans.SrcFilename = "file"

			So(ctx.db.Insert(trans).Run(), ShouldBeNil)

			localPath := filepath.Join(ctx.root, ctx.send.LocalDir, trans.SrcFilename)
			So(fs.WriteFullFile(localPath, []byte("Hello World")), ShouldBeNil)

			pip := newTestPipeline(c, ctx.db, trans)

			So(pip.machine.Transition(statePreTasks), ShouldBeNil)
			So(pip.machine.Transition(statePreTasksDone), ShouldBeNil)

			Convey("When creating a new transfer stream", func(c C) {
				stream, err := newFileStream(pip.Pipeline, false)
				So(err, ShouldBeNil)
				Reset(func() { _ = stream.file.Close() })

				Convey("Then it should  return a new transfer stream", func(c C) {
					So(stream, ShouldNotBeNil)

					Convey("Then the transfer file should have been opened", func(c C) {
						So(stream.file, ShouldNotBeNil)

						info, statErr := stream.Stat()
						So(statErr, ShouldBeNil)
						So(info.Name(), ShouldEqual, path.Base(trans.LocalPath))
					})
				})
			})

			Convey("Given that the file does not exist", func(c C) {
				So(fs.RemoveAll(trans.LocalPath), ShouldBeNil)

				Convey("When creating a new transfer stream", func(c C) {
					_, err := newFileStream(pip.Pipeline, false)

					Convey("Then it should return an error", func(c C) {
						So(err, ShouldBeError, NewError(types.TeFileNotFound,
							"file not found"))
					})
				})
			})
		})

		Convey("Given a new receive transfer", func(c C) {
			trans.RuleID = ctx.recv.ID
			trans.SrcFilename = "file"

			So(ctx.db.Insert(trans).Run(), ShouldBeNil)

			transCtx, err := model.GetTransferContext(ctx.db, ctx.logger, trans)
			So(err, ShouldBeNil)

			pip, err := NewClientPipeline(ctx.db, ctx.logger, transCtx, nil)
			So(err, ShouldBeNil)

			Reset(pip.doneOK)

			So(pip.machine.Transition(statePreTasks), ShouldBeNil)
			So(pip.machine.Transition(statePreTasksDone), ShouldBeNil)

			Convey("When creating a new transfer stream", func(c C) {
				stream, err := newFileStream(pip, false)
				So(err, ShouldBeNil)
				Reset(func() { _ = stream.file.Close() })

				Convey("Then it should return a new transfer stream", func(c C) {
					So(stream, ShouldNotBeNil)

					Convey("Then the transfer file should have been opened", func(c C) {
						So(stream.file, ShouldNotBeNil)

						info, statErr := stream.Stat()
						So(statErr, ShouldBeNil)
						So(info.Name(), ShouldEqual, trans.SrcFilename+".part")
					})
				})
			})
		})
	})
}

func TestStreamRead(t *testing.T) {
	root := t.TempDir()

	Convey("Given an outgoing transfer", t, func(c C) {
		ctx := initTestDB(c, root)

		trans := &model.Transfer{
			ClientID:        utils.NewNullInt64(ctx.client.ID),
			RemoteAccountID: utils.NewNullInt64(ctx.remoteAccount.ID),
			RuleID:          ctx.send.ID,
			SrcFilename:     "file",
		}
		So(ctx.db.Insert(trans).Run(), ShouldBeNil)

		Convey("Given a file stream for this transfer", func(c C) {
			content := []byte("read file content")
			localPath := path.Join(ctx.root, ctx.send.LocalDir, trans.SrcFilename)

			So(fs.WriteFullFile(localPath, content), ShouldBeNil)

			stream := initFilestream(ctx, trans)

			Convey("When reading from the stream", func(c C) {
				b := make([]byte, 4)
				n, err := stream.Read(b)
				So(err, ShouldBeNil)

				Convey("Then it should return the correct number of bytes", func(c C) {
					So(n, ShouldEqual, len(b))
				})

				Convey("Then the transfer progression should have been updated", func(c C) {
					So(stream.TransCtx.Transfer.Progress, ShouldEqual, len(b))
				})

				Convey("Then the array should contain the file content", func(c C) {
					So(string(b), ShouldEqual, string(content[:len(b)]))
				})
			})

			Convey("Given that an error occurs while reading the file", func(c C) {
				addFileError(stream)

				b := make([]byte, 4)
				_, err := stream.Read(b)
				So(err, ShouldBeError, errRead)

				Convey("Then it should have called the error-tasks", func(c C) {
					waitEndTransfer(stream.Pipeline)

					Convey("Then any subsequent call to 'Read' should return the same error", func(c C) {
						_, err := stream.Read(b)
						So(err, ShouldBeError, errRead)
					})
				})
			})

			Convey("Given that database error occurs", func(c C) {
				database.SimulateError(c, ctx.db)
				time.Sleep(testTransferUpdateInterval) // just to be sure the ticker had the time to tick at least once

				b := make([]byte, 4)
				_, err := stream.Read(b)
				So(err, ShouldBeError, errDatabase)

				Convey("Then it should have called the error-tasks", func(c C) {
					waitEndTransfer(stream.Pipeline)

					Convey("Then any subsequent call to 'Read' should return the same error", func(c C) {
						_, err := stream.Read(b)
						So(err, ShouldBeError, errDatabase)
					})
				})
			})
		})
	})
}

func TestStreamReadAt(t *testing.T) {
	root := t.TempDir()

	Convey("Given an outgoing transfer", t, func(c C) {
		ctx := initTestDB(c, root)

		trans := &model.Transfer{
			ClientID:        utils.NewNullInt64(ctx.client.ID),
			RemoteAccountID: utils.NewNullInt64(ctx.remoteAccount.ID),
			RuleID:          ctx.send.ID,
			SrcFilename:     "file",
		}
		So(ctx.db.Insert(trans).Run(), ShouldBeNil)

		Convey("Given a file stream for this transfer", func(c C) {
			content := []byte("read file content")
			localPath := path.Join(ctx.root, ctx.send.LocalDir, trans.SrcFilename)

			So(fs.WriteFullFile(localPath, content), ShouldBeNil)

			stream := initFilestream(ctx, trans)

			Convey("When reading from the stream with an offset", func(c C) {
				b := make([]byte, 4)
				off := 2
				n, err := stream.ReadAt(b, int64(off))
				So(err, ShouldBeNil)

				Convey("Then it should return the correct number of bytes", func(c C) {
					So(n, ShouldEqual, len(b))
				})

				Convey("Then the transfer progression should have been updated", func(c C) {
					So(stream.TransCtx.Transfer.Progress, ShouldEqual, len(b))
				})

				Convey("Then the array should contain the file content starting from the offset", func(c C) {
					So(string(b), ShouldEqual, string(content[off:off+len(b)]))
				})
			})

			Convey("Given that an error occurs while reading the file", func(c C) {
				addFileError(stream)

				b := make([]byte, 4)
				_, err := stream.ReadAt(b, 0)
				So(err, ShouldBeError, errRead)

				Convey("Then it should have called the error-tasks", func(c C) {
					waitEndTransfer(stream.Pipeline)

					Convey("Then any subsequent call to 'ReadAt' should return the same error", func(c C) {
						_, err := stream.ReadAt(b, 0)
						So(err, ShouldBeError, errRead)
					})
				})
			})

			Convey("Given that an error occurs while updating the progress", func(c C) {
				database.SimulateError(c, ctx.db)
				time.Sleep(testTransferUpdateInterval) // just to be sure the ticker had the time to tick

				b := make([]byte, 4)
				_, err := stream.ReadAt(b, 0)
				So(err, ShouldBeError, errDatabase)

				Convey("Then it should have called the error-tasks", func(c C) {
					waitEndTransfer(stream.Pipeline)

					Convey("Then any subsequent call to 'ReadAt' should return the same error", func(c C) {
						_, err := stream.ReadAt(b, 0)
						So(err, ShouldBeError, errDatabase)
					})
				})
			})
		})
	})
}

func TestStreamWrite(t *testing.T) {
	root := t.TempDir()

	Convey("Given an incoming transfer", t, func(c C) {
		ctx := initTestDB(c, root)

		trans := &model.Transfer{
			ClientID:        utils.NewNullInt64(ctx.client.ID),
			RemoteAccountID: utils.NewNullInt64(ctx.remoteAccount.ID),
			RuleID:          ctx.recv.ID,
			SrcFilename:     "file",
		}
		So(ctx.db.Insert(trans).Run(), ShouldBeNil)

		Convey("Given a file stream for this transfer", func(c C) {
			stream := initFilestream(ctx, trans)

			Convey("When writing to the stream", func(c C) {
				b := []byte("file content")
				n, err := stream.Write(b)
				So(err, ShouldBeNil)

				Convey("Then it should return the correct number of bytes", func(c C) {
					So(n, ShouldEqual, len(b))
				})

				Convey("Then the transfer progression should have been updated", func(c C) {
					So(stream.TransCtx.Transfer.Progress, ShouldEqual, len(b))
				})

				Convey("Then the file should contain the array content", func(c C) {
					So(stream.Sync(), ShouldBeNil)

					content, err := fs.ReadFullFile(trans.LocalPath)
					So(err, ShouldBeNil)
					So(string(content), ShouldEqual, string(b))
				})
			})

			Convey("Given that an error occurs while writing the file", func(c C) {
				addFileError(stream)

				b := make([]byte, 4)
				_, err := stream.Write(b)
				So(err, ShouldBeError, errWrite)

				Convey("Then it should have called the error-tasks", func(c C) {
					waitEndTransfer(stream.Pipeline)

					Convey("Then any subsequent call to 'Write' should return the same error", func(c C) {
						_, err := stream.Write(b)
						So(err, ShouldBeError, errWrite)
					})
				})
			})

			Convey("Given that an error occurs while updating the progress", func(c C) {
				database.SimulateError(c, ctx.db)
				time.Sleep(testTransferUpdateInterval) // just to be sure the ticker had the time to tick

				b := make([]byte, 4)
				_, err := stream.Write(b)
				So(err, ShouldBeError, errDatabase)

				Convey("Then it should have called the error-tasks", func(c C) {
					waitEndTransfer(stream.Pipeline)

					Convey("Then any subsequent call to 'Write' should return the same error", func(c C) {
						_, err := stream.Write(b)
						So(err, ShouldBeError, errDatabase)
					})
				})
			})
		})
	})
}

func TestStreamWriteAt(t *testing.T) {
	root := t.TempDir()

	Convey("Given an incoming transfer", t, func(c C) {
		ctx := initTestDB(c, root)

		trans := &model.Transfer{
			ClientID:        utils.NewNullInt64(ctx.client.ID),
			RemoteAccountID: utils.NewNullInt64(ctx.remoteAccount.ID),
			RuleID:          ctx.recv.ID,
			SrcFilename:     "file",
		}
		So(ctx.db.Insert(trans).Run(), ShouldBeNil)

		Convey("Given a file stream for this transfer", func(c C) {
			stream := initFilestream(ctx, trans)

			Convey("When writing to the stream with an offset", func(c C) {
				b := []byte("file content")
				off := 2
				n, err := stream.WriteAt(b, int64(off))
				So(err, ShouldBeNil)

				Convey("Then it should return the correct number of bytes", func(c C) {
					So(n, ShouldEqual, len(b))
				})

				Convey("Then the transfer progression should have been updated", func(c C) {
					So(stream.TransCtx.Transfer.Progress, ShouldEqual, len(b))
				})

				Convey("Then the file should contain the array content with an offset", func(c C) {
					So(stream.file.Close(), ShouldBeNil)
					content, err := fs.ReadFullFile(trans.LocalPath)
					So(err, ShouldBeNil)

					So(string(content), ShouldEqual, strings.Repeat("\000", off)+string(b))
				})
			})

			Convey("Given that an error occurs while writing the file", func(c C) {
				addFileError(stream)

				b := make([]byte, 4)
				_, err := stream.WriteAt(b, 0)
				So(err, ShouldBeError, errWrite)

				Convey("Then it should have called the error-tasks", func(c C) {
					waitEndTransfer(stream.Pipeline)

					Convey("Then any subsequent call to 'WriteAt' should return the same error", func(c C) {
						_, err := stream.WriteAt(b, 0)
						So(err, ShouldBeError, errWrite)
					})
				})
			})

			Convey("Given that an error occurs while updating the progress", func(c C) {
				database.SimulateError(c, ctx.db)
				time.Sleep(testTransferUpdateInterval) // just to be sure the ticker had the time to tick

				b := make([]byte, 4)
				_, err := stream.WriteAt(b, 0)
				So(err, ShouldBeError, errDatabase)

				Convey("Then it should have called the error-tasks", func(c C) {
					waitEndTransfer(stream.Pipeline)

					Convey("Then any subsequent call to 'WriteAt' should return the same error", func(c C) {
						_, err := stream.WriteAt(b, 0)
						So(err, ShouldBeError, errDatabase)
					})
				})
			})
		})
	})
}

func TestStreamClose(t *testing.T) {
	root := t.TempDir()

	Convey("Given an incoming transfer", t, func(c C) {
		ctx := initTestDB(c, root)

		trans := &model.Transfer{
			ClientID:        utils.NewNullInt64(ctx.client.ID),
			RemoteAccountID: utils.NewNullInt64(ctx.remoteAccount.ID),
			RuleID:          ctx.recv.ID,
			SrcFilename:     "file",
		}
		So(ctx.db.Insert(trans).Run(), ShouldBeNil)

		Convey("Given a file stream for this transfer", func(c C) {
			stream := initFilestream(ctx, trans)
			So(stream.machine.Transition(stateDataEnd), ShouldBeNil)

			Convey("When closing the stream", func(c C) {
				So(stream.close(), ShouldBeNil)

				Convey("Then the underlying file should be closed", func(c C) {
					So(stream.file.Close(), ShouldWrap, fs.ErrClosed)
				})
			})

			Convey("Given that an error occurs while updating the progress", func(c C) {
				database.SimulateError(c, ctx.db)
				time.Sleep(testTransferUpdateInterval)
				So(stream.close(), ShouldBeError, errDatabase)

				Convey("Then it should have called the error-tasks", func(c C) {
					waitEndTransfer(stream.Pipeline)
				})
			})
		})
	})
}

func TestStreamMove(t *testing.T) {
	rootRcv := t.TempDir()
	rootSnd := t.TempDir()

	Convey("Given an incoming transfer", t, func(c C) {
		ctx := initTestDB(c, rootRcv)

		trans := &model.Transfer{
			ClientID:        utils.NewNullInt64(ctx.client.ID),
			RemoteAccountID: utils.NewNullInt64(ctx.remoteAccount.ID),
			RuleID:          ctx.recv.ID,
			SrcFilename:     "file",
		}
		So(ctx.db.Insert(trans).Run(), ShouldBeNil)

		Convey("Given a closed file stream for this transfer", func(c C) {
			stream := initFilestream(ctx, trans)
			So(stream.machine.Transition(stateDataEnd), ShouldBeNil)
			So(stream.close(), ShouldBeNil)

			Convey("When moving the file", func(c C) {
				So(stream.move(), ShouldBeNil)

				Convey("Then the underlying file should have been be moved", func(c C) {
					file := path.Join(ctx.root, ctx.recv.LocalDir, "file")
					_, err := fs.Stat(file)
					So(err, ShouldBeNil)
				})
			})

			Convey("Given that the move fails", func(c C) {
				So(fs.RemoveAll(stream.TransCtx.Transfer.LocalPath), ShouldBeNil)

				Convey("When moving the file", func(c C) {
					So(stream.move(), ShouldBeError,
						NewError(types.TeFinalization, "temp file rename failed"))

					Convey("Then it should have called the error tasks", func(c C) {
						waitEndTransfer(stream.Pipeline)
					})
				})
			})

			Convey("Given that a database error occurs", func(c C) {
				database.SimulateError(c, ctx.db)
				time.Sleep(testTransferUpdateInterval)

				Convey("When moving the file", func(c C) {
					So(stream.move(), ShouldBeError, errDatabase)

					Convey("Then it should have called the error tasks", func(c C) {
						waitEndTransfer(stream.Pipeline)
					})
				})
			})
		})
	})

	Convey("Given an outgoing transfer", t, func(c C) {
		ctx := initTestDB(c, rootSnd)

		trans := &model.Transfer{
			ClientID:        utils.NewNullInt64(ctx.client.ID),
			RemoteAccountID: utils.NewNullInt64(ctx.remoteAccount.ID),
			RuleID:          ctx.recv.ID,
			SrcFilename:     "file",
		}
		So(ctx.db.Insert(trans).Run(), ShouldBeNil)

		filePath := fs.JoinPath(ctx.root, ctx.recv.LocalDir, "file")
		So(fs.WriteFullFile(filePath, []byte("file content")), ShouldBeNil)
		// Reset(func() { _ = ctx.fs.Remove(path) })

		Convey("Given a closed file stream for this transfer", func(c C) {
			stream := initFilestream(ctx, trans)
			So(stream.machine.Transition(stateDataEnd), ShouldBeNil)
			So(stream.close(), ShouldBeNil)

			Convey("When moving the file", func(c C) {
				So(stream.move(), ShouldBeNil)

				Convey("Then it should do nothing", func(c C) {
					So(stream.TransCtx.Transfer.LocalPath, ShouldEqual, filePath)
				})
			})
		})
	})
}

func TestStreamSeek(t *testing.T) {
	root := t.TempDir()

	Convey("Given an incoming transfer", t, func(c C) {
		ctx := initTestDB(c, root)

		trans := &model.Transfer{
			RemoteAccountID: utils.NewNullInt64(ctx.remoteAccount.ID),
			ClientID:        utils.NewNullInt64(ctx.client.ID),
			RuleID:          ctx.recv.ID,
			SrcFilename:     "file",
		}
		So(ctx.db.Insert(trans).Run(), ShouldBeNil)

		stream := initFilestream(ctx, trans)

		Convey("When calling the Seek function", func(c C) {
			const seekOffset int64 = 5

			newOff, err := stream.Seek(seekOffset, io.SeekStart)
			So(err, ShouldBeNil)
			So(newOff, ShouldEqual, seekOffset)

			Convey("Then the file offset should have changed", func(c C) {
				off, err := stream.file.Seek(0, io.SeekCurrent)
				So(err, ShouldBeNil)
				So(off, ShouldEqual, seekOffset)
			})

			Convey("Then the transfer progression should have changed", func(c C) {
				<-time.After(testTransferUpdateInterval)

				So(stream.UpdateTrans(), ShouldBeNil)

				var check model.Transfer

				So(ctx.db.Get(&check, "id=?", trans.ID).Run(), ShouldBeNil)
				So(check.Progress, ShouldEqual, seekOffset)
			})
		})
	})
}

func TestStreamSync(t *testing.T) {
	root := t.TempDir()

	Convey("Given an incoming transfer", t, func(c C) {
		ctx := initTestDB(c, root)

		trans := &model.Transfer{
			RemoteAccountID: utils.NewNullInt64(ctx.remoteAccount.ID),
			ClientID:        utils.NewNullInt64(ctx.client.ID),
			RuleID:          ctx.recv.ID,
			SrcFilename:     "file",
		}
		So(ctx.db.Insert(trans).Run(), ShouldBeNil)

		stream := initFilestream(ctx, trans)
		stream.updTicker.Reset(time.Hour)
		So(stream.UpdateTrans(), ShouldBeNil)

		select {
		case <-stream.updTicker.C:
		default:
		}

		bytes := []byte("foobar")
		_, err := stream.Write(bytes)
		So(err, ShouldBeNil)

		var checkBefore model.Transfer

		So(ctx.db.Get(&checkBefore, "id=?", trans.ID).Run(), ShouldBeNil)
		So(checkBefore.Progress, ShouldEqual, 0)

		Convey("When calling the Sync function", func(c C) {
			So(stream.Sync(), ShouldBeNil)

			Convey("Then the content should have written to storage", func(c C) {
				content, err := fs.ReadFullFile(trans.LocalPath)
				So(err, ShouldBeNil)
				So(content, ShouldResemble, bytes)
			})

			Convey("Then the transfer progression should have changed", func(c C) {
				var checkAfter model.Transfer

				So(ctx.db.Get(&checkAfter, "id=?", trans.ID).Run(), ShouldBeNil)
				So(checkAfter.Progress, ShouldEqual, len(bytes))
			})

			Convey("Then we can still write to the stream", func(c C) {
				b2 := []byte("extra file content")
				_, err := stream.Write(b2)
				So(err, ShouldBeNil)
			})
		})
	})
}
