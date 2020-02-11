// Package pipeline regroups all the types and interfaces used in transfer pipelines.
package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tasks"
)

func checkSignal(ch <-chan model.Signal) *model.PipelineError {
	select {
	case signal := <-ch:
		switch signal {
		case model.SignalCancel:
			return &model.PipelineError{Kind: model.KindCancel}
		case model.SignalPause:
			return &model.PipelineError{Kind: model.KindPause}
		case model.SignalShutdown:
			return &model.PipelineError{Kind: model.KindInterrupt}
		default:
			return nil
		}
	default:
		return nil
	}
}

func createTransfer(logger *log.Logger, db *database.Db, trans *model.Transfer) error {
	err := db.Create(trans)
	if err != nil {
		logger.Criticalf("Failed to create transfer entry: %s", err)
		return err
	}
	return nil
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
	step model.TransferStep) *model.PipelineError {

	proc.Transfer.Step = step
	if err := proc.Transfer.Update(proc.Db); err != nil {
		proc.Logger.Criticalf("Failed to update transfer status: %s", err)
		return &model.PipelineError{Kind: model.KindDatabase}
	}

	tasksList, err := proc.GetTasks(chain)
	if err != nil {
		proc.Logger.Criticalf("Failed to retrieve tasks: %s", err)
		return err
	}

	return proc.RunTasks(tasksList)
}

func getFile(logger *log.Logger, root string, rule *model.Rule,
	trans *model.Transfer) (*os.File, *model.PipelineError) {

	if rule.IsSend {
		path := filepath.Clean(filepath.Join(root, rule.Path, trans.SourcePath))
		file, err := os.Open(path)
		if err != nil {
			logger.Errorf("Failed to open source file: %s", err)
			return nil, model.NewPipelineError(model.TeForbidden, err.Error())
		}
		return file, nil
	}

	path := filepath.Clean(filepath.Join(root, rule.Path, trans.DestPath))
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		logger.Errorf("Failed to create destination file: %s", err)
		return nil, model.NewPipelineError(model.TeForbidden, err.Error())
	}
	return file, nil
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

// HandleError analyses the given error, and executes the necessary steps
// corresponding to the error kind.
func HandleError(stream *TransferStream, err *model.PipelineError) {
	switch err.Kind {
	case model.KindCancel:
		stream.Transfer.Status = model.StatusCancelled
		stream.Archive()
	case model.KindDatabase:
	case model.KindInterrupt:
		stream.Transfer.Status = model.StatusInterrupted
		if dbErr := stream.Transfer.Update(stream.Db); dbErr != nil {
			stream.Logger.Criticalf("Failed to update transfer error: %s", dbErr)
			return
		}
	case model.KindPause:
		stream.Transfer.Status = model.StatusPaused
		if dbErr := stream.Transfer.Update(stream.Db); dbErr != nil {
			stream.Logger.Criticalf("Failed to update transfer error: %s", dbErr)
			return
		}
	case model.KindTransfer:
		stream.Transfer.Error = err.Cause
		if dbErr := stream.Transfer.Update(stream.Db); dbErr != nil {
			stream.Logger.Criticalf("Failed to update transfer error: %s", dbErr)
			return
		}
		stream.ErrorTasks()
		stream.Transfer.Status = model.StatusError
		stream.Archive()
	}
	stream.Exit()
}
