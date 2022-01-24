// Package tasks regroups all the different types of transfer tasks runners.
package tasks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// Runner provides a way to execute tasks given a transfer context (rule, transfer).
type Runner struct {
	db       *database.DB
	logger   *log.Logger
	transCtx *model.TransferContext
	ctx      context.Context
	Stop     context.CancelFunc
	lock     sync.WaitGroup
}

// NewTaskRunner returns a new tasks.Runner using the given elements.
func NewTaskRunner(db *database.DB, logger *log.Logger, transCtx *model.TransferContext) *Runner {
	ctx, cancel := context.WithCancel(context.Background())
	r := &Runner{
		db:       db,
		logger:   logger,
		transCtx: transCtx,
		ctx:      ctx,
	}

	r.Stop = func() {
		cancel()
		r.lock.Wait()
	}

	return r
}

// PreTasks executes the transfer's pre-tasks.
func (r *Runner) PreTasks() *types.TransferError {
	return r.runTasks(r.transCtx.PreTasks, false)
}

// PostTasks executes the transfer's post-tasks.
func (r *Runner) PostTasks() *types.TransferError {
	return r.runTasks(r.transCtx.PostTasks, false)
}

// ErrorTasks executes the transfer's error-tasks.
func (r *Runner) ErrorTasks() *types.TransferError {
	return r.runTasks(r.transCtx.ErrTasks, true)
}

func (r *Runner) runTask(task model.Task, taskInfo string, isErrTasks bool) *types.TransferError {
	runner, ok := model.ValidTasks[task.Type]
	if !ok {
		return types.NewTransferError(types.TeExternalOperation, fmt.Sprintf(
			"%s: unknown task", taskInfo))
	}

	args, err := r.setup(&task)
	if err != nil {
		logMsg := fmt.Sprintf("%s: %s", taskInfo, err.Error())
		r.logger.Error(logMsg)

		return types.NewTransferError(types.TeExternalOperation, logMsg)
	}

	if validator, ok := runner.(model.TaskValidator); ok {
		if err2 := validator.Validate(args); err2 != nil {
			logMsg := fmt.Sprintf("%s: %s", taskInfo, err2.Error())
			r.logger.Error(logMsg)

			return types.NewTransferError(types.TeExternalOperation, logMsg)
		}
	}

	var msg string
	if isErrTasks {
		msg, err = runner.Run(context.Background(), args, r.db, r.transCtx)
	} else {
		msg, err = runner.Run(r.ctx, args, r.db, r.transCtx)
	}

	if err != nil {
		errMsg := fmt.Sprintf("%s: %s", taskInfo, err)

		var warning *warningError
		if !errors.As(err, &warning) {
			r.logger.Error(errMsg)
			r.transCtx.Transfer.Error = *types.NewTransferError(types.TeExternalOperation, errMsg)

			return &r.transCtx.Transfer.Error
		}

		r.logger.Warning(errMsg)
		r.transCtx.Transfer.Error = *types.NewTransferError(types.TeWarning, errMsg)

		if dbErr := r.db.Update(r.transCtx.Transfer).Cols("error_code", "error_details").Run(); dbErr != nil {
			r.logger.Errorf("Failed to update task status: %s", dbErr)

			if !isErrTasks {
				return types.NewTransferError(types.TeInternal, "database error")
			}
		}
	} else {
		if msg != "" {
			r.logger.Debugf("%s: %s", taskInfo, msg)
		} else {
			r.logger.Debug(taskInfo)
		}
	}

	return r.updateProgress(isErrTasks)
}

func (r *Runner) updateProgress(isErrTasks bool) *types.TransferError {
	r.transCtx.Transfer.TaskNumber++
	query := r.db.Update(r.transCtx.Transfer).Cols("task_number")

	size := r.getFilesize()
	if size >= 0 && size != r.transCtx.Transfer.Filesize {
		r.transCtx.Transfer.Filesize = size

		query.Cols("filesize")
	}

	if dbErr := query.Run(); dbErr != nil {
		r.logger.Errorf("Failed to update task number: %s", dbErr)

		if !isErrTasks {
			return types.NewTransferError(types.TeInternal, "database error")
		}
	}

	return nil
}

// runTasks executes sequentially the list of tasks given according to the
// Runner context.
func (r *Runner) runTasks(tasks []model.Task, isErrTasks bool) *types.TransferError {
	r.lock.Add(1)
	defer r.lock.Done()

	for i := r.transCtx.Transfer.TaskNumber; i < uint64(len(tasks)); i++ {
		task := tasks[i]
		taskInfo := fmt.Sprintf("Task %s @ %s %s[%v]", task.Type, r.transCtx.Rule.Name,
			task.Chain, task.Rank)

		if !isErrTasks {
			select {
			case <-r.ctx.Done():
				return types.NewTransferError(types.TeInternal, "transfer interrupted")
			default:
			}
		}

		if err := r.runTask(task, taskInfo, isErrTasks); err != nil {
			return err
		}
	}

	return nil
}

// setup contextualizes and unmarshalls the tasks arguments.
// It returns a json object exploitable by the task.
func (r *Runner) setup(t *model.Task) (map[string]string, error) {
	sArgs, err := r.replace(t)
	if err != nil {
		return nil, err
	}

	args := map[string]string{}
	if err := json.Unmarshal(sArgs, &args); err != nil {
		return nil, fmt.Errorf("cannot parse task arguments: %w", err)
	}

	return args, nil
}

// replace replace all the context variables (#varname#) in the tasks arguments
// by their context value.
func (r *Runner) replace(t *model.Task) ([]byte, error) {
	res := t.Args
	for key, f := range replacers {
		if bytes.Contains(res, []byte(key)) {
			r, err := f(r)
			if err != nil {
				return nil, err
			}

			rep, err := json.Marshal(r)
			if err != nil {
				return nil, fmt.Errorf("cannot prepare value for replacement: %w", err)
			}

			rep = rep[1 : len(rep)-1]
			res = bytes.ReplaceAll(res, []byte(key), rep)
		}
	}

	return res, nil
}

func (r *Runner) getFilesize() int64 {
	if !r.transCtx.Rule.IsSend {
		return -1
	}

	info, err := os.Stat(r.transCtx.Transfer.LocalPath)
	if err != nil {
		r.logger.Warningf("Failed to retrieve file size: %s", err)

		return -1
	}

	return info.Size()
}
