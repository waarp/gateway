// Package pipeline regroups all the types and interfaces used in transfer pipelines.
package pipeline

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tasks"
)

// Paths is a struct combining the paths given in the Gateway configuration file
// and the root of a server.
type Paths struct {
	conf.PathsConfig
	ServerRoot string
}

func checkSignal(ctx context.Context, ch <-chan model.Signal) *model.PipelineError {
	select {
	case <-ctx.Done():
		return &model.PipelineError{Kind: model.KindInterrupt}
	case signal := <-ch:
		switch signal {
		case model.SignalCancel:
			return &model.PipelineError{Kind: model.KindCancel}
		case model.SignalPause:
			return &model.PipelineError{Kind: model.KindPause}
		default:
			return nil
		}
	default:
		return nil
	}
}

func createTransfer(logger *log.Logger, db *database.DB, trans *model.Transfer) *model.PipelineError {
	if err := db.Create(trans); err != nil {
		if _, ok := err.(*database.ErrInvalid); ok {
			logger.Errorf("Failed to create transfer entry: %s", err.Error())
			return model.NewPipelineError(model.TeForbidden, err.Error())
		}
		logger.Criticalf("Failed to create transfer entry: %s", err.Error())
		return &model.PipelineError{Kind: model.KindDatabase}
	}
	return nil
}

// ToHistory removes the given transfer from the database, converts it into a
// history entry, and inserts the new history entry in the database.
// If any of these steps fails, the changes are reverted and an error is returned.
func ToHistory(db *database.DB, logger *log.Logger, trans *model.Transfer) error {
	ses, err := db.BeginTransaction()
	if err != nil {
		logger.Criticalf("Failed to start archival transaction: %s", err)
		return err
	}

	if err := ses.Delete(&model.Transfer{ID: trans.ID}); err != nil {
		logger.Criticalf("Failed to delete transfer for archival: %s", err)
		ses.Rollback()
		return err
	}

	if trans.Error.Code == model.TeOk || trans.Error.Code == model.TeWarning {
		trans.Progress = 0
		trans.Step = ""
	}

	hist, err := trans.ToHistory(ses, time.Now())
	if err != nil {
		logger.Criticalf("Failed to convert transfer to history: %s", err)
		ses.Rollback()
		return err
	}

	if err := ses.Create(hist); err != nil {
		logger.Criticalf("Failed to create new history entry: %s", err)
		ses.Rollback()
		return err
	}

	if err := ses.Commit(); err != nil {
		logger.Criticalf("Failed to commit archival transaction: %s", err)
		return err
	}
	return nil
}

func execTasks(proc *tasks.Processor, chain model.Chain,
	step model.TransferStep) *model.PipelineError {

	proc.Transfer.Step = step
	if err := proc.Transfer.Update(proc.DB); err != nil {
		proc.Logger.Criticalf("Failed to update transfer step: %s", err)
		return &model.PipelineError{Kind: model.KindDatabase}
	}

	tasksList, err := proc.GetTasks(chain)
	if err != nil {
		proc.Logger.Criticalf("Failed to retrieve tasks: %s", err)
		return err
	}

	return proc.RunTasks(tasksList)
}

func getFile(logger *log.Logger, rule *model.Rule, trans *model.Transfer) (*os.File, *model.PipelineError) {

	path := trans.TrueFilepath
	if rule.IsSend {
		file, err := os.OpenFile(path, os.O_RDONLY, 0100)
		if err != nil {
			logger.Errorf("Failed to open source file: %s", err)
			return nil, model.NewPipelineError(model.TeForbidden, err.Error())
		}
		if trans.Progress != 0 {
			if _, err := file.Seek(int64(trans.Progress), io.SeekStart); err != nil {
				logger.Errorf("Failed to seek inside file: %s", err)
				return nil, model.NewPipelineError(model.TeForbidden, err.Error())
			}
		}
		return file, nil
	}

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		logger.Errorf("Failed to create destination file (%s): %s", path, err)
		return nil, model.NewPipelineError(model.TeForbidden, err.Error())
	}
	if trans.Progress != 0 {
		if _, err := file.Seek(int64(trans.Progress), io.SeekStart); err != nil {
			logger.Errorf("Failed to seek inside file: %s", err)
			return nil, model.NewPipelineError(model.TeForbidden, err.Error())
		}
	}
	return file, nil
}

func makeDir(path string) error {
	dir := filepath.Dir(path)
	fmt.Printf("MAKE DIR => %s\n", path)
	if info, err := os.Lstat(dir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0740); err != nil {
				return err
			}
		} else {
			return err
		}
	} else if !info.IsDir() {
		return fmt.Errorf("a file named '%s' already exist", filepath.Base(dir))
	}
	return nil
}

// HandleError analyses the given error, and executes the necessary steps
// corresponding to the error kind.
func HandleError(stream *TransferStream, err *model.PipelineError) {
	_ = stream.Close()

	switch err.Kind {
	case model.KindCancel:
		stream.Transfer.Status = model.StatusCancelled
		stream.Archive()
		_ = os.Remove(stream.File.Name())
	case model.KindDatabase:
	case model.KindInterrupt:
		stream.Transfer.Status = model.StatusInterrupted
		if dbErr := stream.Transfer.Update(stream.DB); dbErr != nil {
			stream.Logger.Criticalf("Failed to update transfer error: %s", dbErr)
			return
		}
	case model.KindPause:
		stream.Transfer.Status = model.StatusPaused
		if dbErr := stream.Transfer.Update(stream.DB); dbErr != nil {
			stream.Logger.Criticalf("Failed to update transfer error: %s", dbErr)
			return
		}
	case model.KindTransfer:
		stream.Transfer.Error = err.Cause
		if dbErr := stream.Transfer.Update(stream.DB); dbErr != nil {
			stream.Logger.Criticalf("Failed to update transfer error: %s", dbErr)
			return
		}
		stream.ErrorTasks()
		stream.Transfer.Error = err.Cause
		if dbErr := stream.Transfer.Update(stream.DB); dbErr != nil {
			stream.Logger.Criticalf("Failed to update transfer step: %s", dbErr)
			return
		}
		stream.Transfer.Status = model.StatusError
		stream.Archive()
	}
	stream.Exit()
}
