package sftp

import (
	"io"
	"os"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tasks"
)

type transferStream struct {
	db       *database.Db
	logger   *log.Logger
	file     *os.File
	trans    *model.Transfer
	rule     *model.Rule
	shutdown <-chan bool

	fail  model.TransferError
	bytes uint64
}

func (t *transferStream) postTasks(proc *tasks.Processor) model.TransferError {
	if err := t.db.Update(&model.Transfer{Status: model.StatusPostTasks},
		t.trans.ID, false); err != nil {
		return model.NewTransferError(model.TeInternal, err.Error())
	}

	postTasks, err := proc.GetTasks(model.ChainPost)
	if err != nil {
		t.logger.Criticalf("Failed to retrieve transfer PostTasks: %s", err)
		return model.NewTransferError(model.TeInternal, err.Error())
	}

	return proc.RunTasks(postTasks)
}

func (t *transferStream) errorTasks(proc *tasks.Processor) {
	stat := &model.Transfer{
		Status: model.StatusPostTasks,
		Error:  t.fail,
	}

	if err := t.db.Update(stat, t.trans.ID, false); err != nil {
		t.logger.Criticalf("Failed to update transfer status: %s", err)
		return
	}

	errTasks, err := proc.GetTasks(model.ChainError)
	if err != nil {
		t.logger.Criticalf("Failed to retrieve transfer ErrorTasks: %s", err)
		return
	}

	_ = proc.RunTasks(errTasks)
	t.trans.Status = model.StatusDone
}

func (t *transferStream) close() {
	proc := &tasks.Processor{
		Db:       t.db,
		Logger:   t.logger,
		Rule:     t.rule,
		Transfer: t.trans,
	}

	if t.fail.Code == model.TeOk {
		if t.fail = t.postTasks(proc); t.fail.Code == model.TeOk {
			t.toHistory(model.StatusDone)
			return
		}
	}

	t.errorTasks(proc)
	t.toHistory(model.StatusError)
}

func (t *transferStream) toHistory(status model.TransferStatus) {
	trans := &model.Transfer{ID: t.trans.ID}
	if err := t.db.Get(trans); err != nil {
		t.logger.Errorf("Error retrieving transfer entry: %s", err)
		return
	}

	ses, err := t.db.BeginTransaction()
	if err != nil {
		t.logger.Errorf("Error starting transaction: %s", err)
		return
	}
	if err := ses.Delete(&model.Transfer{ID: t.trans.ID}); err != nil {
		t.logger.Errorf("Error deleting the old transfer: %s", err)
		ses.Rollback()
		return
	}

	trans.Status = status
	hist, err := trans.ToHistory(t.db, time.Now().UTC())
	if err != nil {
		t.logger.Errorf("Error converting transfer to history: %s", err)
		ses.Rollback()
		return
	}

	if err := ses.Create(hist); err != nil {
		t.logger.Errorf("Error inserting new history entry: %s", err)
		ses.Rollback()
		return
	}

	if err := ses.Commit(); err != nil {
		t.logger.Errorf("Error committing the transaction: %s", err)
		return
	}
}

func (t *transferStream) transferError(err error) {
	if err == io.EOF {
		select {
		case <-t.shutdown:
			t.fail = model.NewTransferError(model.TeShuttingDown,
				"SFTP server shutdown initiated")
		default:
			t.fail = model.NewTransferError(model.TeConnectionReset,
				"SFTP connection closed unexpectedly")
		}

	}
	t.fail = model.NewTransferError(model.TeDataTransfer, err.Error())
}

type downloadStream struct {
	*transferStream
}

func (d *downloadStream) WriteAt(p []byte, off int64) (n int, err error) {
	n, err = d.file.WriteAt(p, off)
	if err != nil {
		d.fail = model.NewTransferError(model.TeInternal, err.Error())
	}
	d.transferStream.bytes += uint64(n)
	//TODO: update transfer progress in database

	return
}

func (d *downloadStream) TransferError(err error) {
	d.transferError(err)
}

func (d *downloadStream) Close() error {
	d.close()
	return nil
}

type uploadStream struct {
	*transferStream
}

func (u *uploadStream) ReadAt(p []byte, off int64) (n int, err error) {
	n, err = u.file.ReadAt(p, off)
	if err != nil && err != io.EOF {
		u.fail = model.NewTransferError(model.TeInternal, err.Error())
	}
	u.transferStream.bytes += uint64(n)
	//TODO: update transfer progress in database

	return
}

func (u *uploadStream) TransferError(err error) {
	u.transferError(err)
}

func (u *uploadStream) Close() error {
	u.close()
	return nil
}
