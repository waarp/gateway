// Package tasks regroups all the different types of transfer tasks runners.
package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// Runner provides a way to execute tasks given a transfer context (rule, transfer).
type Runner struct {
	db       *database.DB
	logger   *log.Logger
	transCtx *model.TransferContext
	Ctx      context.Context
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
		Ctx:      ctx,
	}

	r.Stop = func() {
		cancel()
		r.lock.Wait()
	}

	return r
}

// PreTasks executes the transfer's pre-tasks.
func (r *Runner) PreTasks(trace func(rank int8) error) *types.TransferError {
	return r.runTasks(r.transCtx.PreTasks, false, trace)
}

// PostTasks executes the transfer's post-tasks.
func (r *Runner) PostTasks(trace func(rank int8) error) *types.TransferError {
	return r.runTasks(r.transCtx.PostTasks, false, trace)
}

// ErrorTasks executes the transfer's error-tasks.
func (r *Runner) ErrorTasks(trace func(rank int8)) *types.TransferError {
	return r.runTasks(r.transCtx.ErrTasks, true, func(rank int8) error {
		if trace != nil {
			trace(rank)
		}

		return nil
	})
}

func (r *Runner) runTask(task *model.Task, taskInfo string, isErrTasks bool) *types.TransferError {
	runner, ok := model.ValidTasks[task.Type]
	if !ok {
		return types.NewTransferError(types.TeExternalOperation,
			"%s: unknown task", taskInfo)
	}

	args, err := r.setup(task)
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

	if isErrTasks {
		err = runner.Run(context.Background(), args, r.db, r.logger, r.transCtx)
	} else {
		err = runner.Run(r.Ctx, args, r.db, r.logger, r.transCtx)
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
			r.logger.Error("Failed to update task status: %s", dbErr)

			if !isErrTasks {
				return types.NewTransferError(types.TeInternal, "database error")
			}
		}
	} else {
		r.logger.Debug(taskInfo)
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
		r.logger.Error("Failed to update task number: %s", dbErr)

		if !isErrTasks {
			return types.NewTransferError(types.TeInternal, "database error")
		}
	}

	return nil
}

// runTasks executes sequentially the list of tasks given according to the
// Runner context.
func (r *Runner) runTasks(tasks []*model.Task, isErrTasks bool, trace func(rank int8) error,
) *types.TransferError {
	r.lock.Add(1)
	defer r.lock.Done()

	for i := r.transCtx.Transfer.TaskNumber; i < int8(len(tasks)); i++ {
		task := tasks[i]
		taskInfo := fmt.Sprintf("Task %s @ %s %s[%v]", task.Type, r.transCtx.Rule.Name,
			task.Chain, task.Rank)

		if !isErrTasks {
			select {
			case <-r.Ctx.Done():
				return types.NewTransferError(types.TeInternal, "transfer interrupted")
			default:
			}
		}

		if err := r.runTask(task, taskInfo, isErrTasks); err != nil {
			return err
		}

		if trace != nil {
			if err := trace(i); err != nil {
				return testError(err)
			}
		}
	}

	return nil
}

// setup contextualizes and unmarshalls the tasks arguments.
// It returns a json object exploitable by the task.
func (r *Runner) setup(t *model.Task) (map[string]string, error) {
	args, err := r.replace(t)
	if err != nil {
		return nil, err
	}

	return args, nil
}

// replaces all the context variables (#varname#) in the tasks arguments
// by their context value.
func (r *Runner) replace(t *model.Task) (map[string]string, error) {
	raw, jsonErr := json.Marshal(t.Args)
	if jsonErr != nil {
		return nil, fmt.Errorf("failed to serialize the task arguments: %w", jsonErr)
	}

	rawArgs := string(raw)
	replacers := getReplacers()
	replacers.addInfo(r.transCtx)

	for key, f := range replacers {
		if strings.Contains(rawArgs, key) {
			rep, err := f(r)
			if err != nil {
				return nil, err
			}

			bytesRep, err := json.Marshal(rep)
			if err != nil {
				return nil, fmt.Errorf("cannot prepare value for replacement: %w", err)
			}

			replacement := string(bytesRep[1 : len(bytesRep)-1])
			rawArgs = strings.ReplaceAll(rawArgs, key, replacement)
		}
	}

	var newArgs map[string]string
	if err := json.Unmarshal([]byte(rawArgs), &newArgs); err != nil {
		return nil, fmt.Errorf("failed to deserialize the task arguments: %w", err)
	}

	return newArgs, nil
}

func (r *Runner) getFilesize() int64 {
	if !r.transCtx.Rule.IsSend {
		return -1
	}

	info, err := fs.Stat(r.transCtx.FS, &r.transCtx.Transfer.LocalPath)
	if err != nil {
		r.logger.Warning("Failed to retrieve file size: %s", err)

		return -1
	}

	return info.Size()
}
