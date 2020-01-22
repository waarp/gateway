// Package pipeline regroups all the types and interfaces used in transfer pipelines.
package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tasks"
)

// Signals is a map regrouping the signal channels of all ongoing transfers.
// The signal channel of a specific transfer can be retrieved from this map
// using the transfer's ID.
var Signals sync.Map

func createTransfer(logger *log.Logger, db *database.Db, trans *model.Transfer) error {
	err := db.Create(trans)
	if err != nil {
		logger.Criticalf("Failed to create transfer entry: %s", err)
		return err
	}
	return err
}

func toHistory(db *database.Db, logger *log.Logger, trans *model.Transfer) {
	ses, err := db.BeginTransaction()
	if err != nil {
		logger.Criticalf("Failed to start archival transaction: %s", err)
		return
	}

	if err := ses.Delete(&model.Transfer{ID: trans.ID}); err != nil {
		logger.Criticalf("Failed to delete transfer for archival: %s", err)
		ses.Rollback()
		return
	}

	if trans.Error.Code != model.TeOk && trans.Error.Code != model.TeWarning {
		trans.Status = model.StatusError
	} else {
		trans.Status = model.StatusDone
	}

	hist, err := trans.ToHistory(ses, time.Now())
	if err != nil {
		logger.Criticalf("Failed to convert transfer to history: %s", err)
		ses.Rollback()
		return
	}

	if err := ses.Create(hist); err != nil {
		logger.Criticalf("Failed to create new history entry: %s", err)
		ses.Rollback()
		return
	}

	if err := ses.Commit(); err != nil {
		logger.Criticalf("Failed to commit archival transaction: %s", err)
		return
	}
}

func execTasks(proc *tasks.Processor, chain model.Chain,
	status model.TransferStatus) model.TransferError {

	proc.Transfer.Status = status
	if err := proc.Transfer.Update(proc.Db); err != nil {
		proc.Logger.Criticalf("Failed to update transfer status: %s", err)
		return model.NewTransferError(model.TeInternal, err.Error())
	}

	tasksList, err := proc.GetTasks(chain)
	if err != nil {
		proc.Logger.Criticalf("Failed to retrieve tasks: %s", err)
		return model.NewTransferError(model.TeInternal, err.Error())
	}

	return proc.RunTasks(tasksList)
}

func getFile(logger *log.Logger, root string, rule *model.Rule,
	trans *model.Transfer) (*os.File, model.TransferError) {

	if rule.IsSend {
		path := filepath.Clean(filepath.Join(root, rule.Path, trans.SourcePath))
		file, err := os.Open(path)
		if err != nil {
			logger.Errorf("Failed to open source file: %s", err)
			return nil, model.NewTransferError(model.TeForbidden, err.Error())
		}
		return file, model.TransferError{}
	}

	path := filepath.Clean(filepath.Join(root, rule.Path, trans.DestPath))
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		logger.Errorf("Failed to create destination file: %s", err)
		return nil, model.NewTransferError(model.TeForbidden, err.Error())
	}
	return file, model.TransferError{}
}

func makeDir(root, path string) error {
	dir := filepath.FromSlash(fmt.Sprintf("%s/%s", root, path))
	if info, err := os.Lstat(dir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0740); err != nil {
				return err
			}
		} else {
			return err
		}
	} else if !info.IsDir() {
		if err := os.MkdirAll(dir, 0740); err != nil {
			return err
		}
	}
	return nil
}
