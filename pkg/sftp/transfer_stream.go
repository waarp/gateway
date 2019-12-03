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

	fail  error
	bytes uint64
}

func (t *transferStream) endTasks() error {
	proc := &tasks.Processor{
		Db:       t.db,
		Logger:   t.logger,
		Rule:     t.rule,
		Transfer: t.trans,
	}

	if t.fail == nil {
		if err := t.db.Update(&model.Transfer{Status: model.StatusPostTasks},
			t.trans.ID, false); err != nil {
			return err
		}

		postTasks, err := tasks.GetTasks(t.db, t.rule.ID, model.ChainPost)
		if err != nil {
			return err
		}

		t.fail = proc.RunTasks(postTasks)
		if t.fail == nil {
			t.trans.Status = model.StatusDone
			return nil
		}
	}

	stat := &model.Transfer{
		Status: model.StatusErrorTasks,
		Error:  t.trans.Error,
	}
	if err := t.db.Update(stat, t.trans.ID, false); err != nil {
		return err
	}

	errTasks, err := tasks.GetTasks(t.db, t.rule.ID, model.ChainError)
	if err != nil {
		return err
	}

	_ = proc.RunTasks(errTasks)
	t.trans.Status = model.StatusError

	return nil
}

func (t *transferStream) toHistory() {
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

	trans.Status = t.trans.Status
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
			t.trans.Error = model.NewTransferError(model.TeShuttingDown,
				"SFTP server shutdown initiated")
		default:
			t.trans.Error = model.NewTransferError(model.TeConnectionReset,
				"SFTP connection closed unexpectedly")
		}

	}
	t.fail = err
}

type downloadStream struct {
	*transferStream
}

func (d *downloadStream) WriteAt(p []byte, off int64) (n int, err error) {
	n, err = d.file.WriteAt(p, off)
	if err != nil {
		d.fail = err
	}
	d.bytes += uint64(n)
	//TODO: update transfer progress in database

	return
}

func (d *downloadStream) TransferError(err error) {
	d.transferError(err)
}

func (d *downloadStream) Close() error {
	if err := d.endTasks(); err != nil {
		return err
	}
	d.toHistory()
	return nil
}

type uploadStream struct {
	*transferStream
}

func (u *uploadStream) ReadAt(p []byte, off int64) (n int, err error) {
	n, err = u.file.ReadAt(p, off)
	if err != nil && err != io.EOF {
		u.fail = err
	}
	u.bytes += uint64(n)
	//TODO: update transfer progress in database

	return
}

func (u *uploadStream) TransferError(err error) {
	u.transferError(err)
}

func (u *uploadStream) Close() error {
	if err := u.endTasks(); err != nil {
		return err
	}
	u.toHistory()
	return nil
}
