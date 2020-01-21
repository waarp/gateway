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

// ToHistory retrieves a transfer entry from the database and moves it to the
// history table.
func ToHistory(db *database.Db, logger *log.Logger, id uint64, isError bool) {
	trans := &model.Transfer{ID: id}

	if err := db.Get(trans); err != nil {
		logger.Criticalf("Failed to retrieve transfer for archival: %s", err)
		return
	}

	ses, err := db.BeginTransaction()
	if err != nil {
		logger.Criticalf("Failed to start archival transaction: %s", err)
		return
	}

	if err := ses.Delete(&model.Transfer{ID: id}); err != nil {
		logger.Criticalf("Failed to delete transfer for archival: %s", err)
		ses.Rollback()
		return
	}

	if isError {
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

func execTasks(db *database.Db, logger *log.Logger, proc *tasks.Processor,
	chain model.Chain, status model.TransferStatus) model.TransferError {

	stat := &model.Transfer{Status: status}
	if err := db.Update(stat, proc.Transfer.ID, false); err != nil {
		logger.Criticalf("Failed to update transfer status: %s", err)
		return model.NewTransferError(model.TeInternal, err.Error())
	}

	tasksList, err := proc.GetTasks(chain)
	if err != nil {
		logger.Criticalf("Failed to retrieve tasks: %s", err)
		return model.NewTransferError(model.TeInternal, err.Error())
	}

	taskErr := proc.RunTasks(tasksList)
	if taskErr.Code != model.TeOk {
		dbErr := db.Update(&model.Transfer{Error: taskErr}, proc.Transfer.ID, false)
		if dbErr != nil {
			logger.Criticalf("Failed to update transfer error: %s", dbErr)
			return model.NewTransferError(model.TeInternal, dbErr.Error())
		}
	}
	return taskErr
}

// PreTasks retrieves and executes the given processor's error tasks. If an error
// occurs during the execution, a non 'Ok' TransferError is returned.
func PreTasks(db *database.Db, logger *log.Logger, proc *tasks.Processor) model.TransferError {
	return execTasks(db, logger, proc, model.ChainPre, model.StatusPreTasks)
}

// PostTasks retrieves and executes the given processor's post-tasks. If an error
// occurs during the execution, a non 'Ok' TransferError is returned
func PostTasks(db *database.Db, logger *log.Logger, proc *tasks.Processor) model.TransferError {
	return execTasks(db, logger, proc, model.ChainPost, model.StatusPostTasks)
}

// ErrorTasks retrieves and executes the given processor's error tasks.
func ErrorTasks(db *database.Db, logger *log.Logger, proc *tasks.Processor,
	te model.TransferError) model.TransferError {

	stat := &model.Transfer{Status: model.StatusErrorTasks, Error: te}
	if err := db.Update(stat, proc.Transfer.ID, false); err != nil {
		logger.Criticalf("Failed to update transfer status: %s", err)
		return model.NewTransferError(model.TeInternal, err.Error())
	}

	errorTasks, err := proc.GetTasks(model.ChainError)
	if err != nil {
		logger.Criticalf("Failed to retrieve error-tasks: %s", err)
		return model.NewTransferError(model.TeInternal, err.Error())
	}

	return proc.RunTasks(errorTasks)
}

// MakeDir recursively creates a new directory by navigating to the given path,
// starting from the given root. If the directory already exist, this function
// does nothing. An error is returned if the directory cannot be created.
func MakeDir(root, path string) error {
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
