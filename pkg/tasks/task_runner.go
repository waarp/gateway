// Package tasks regroups all the different types of transfer tasks runners.
package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

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
func (r *Runner) PreTasks(trace func(rank int8) error) *Error {
	return r.runTasks(r.transCtx.PreTasks, false, trace)
}

// PostTasks executes the transfer's post-tasks.
func (r *Runner) PostTasks(trace func(rank int8) error) *Error {
	return r.runTasks(r.transCtx.PostTasks, false, trace)
}

// ErrorTasks executes the transfer's error-tasks.
func (r *Runner) ErrorTasks(trace func(rank int8)) *Error {
	return r.runTasks(r.transCtx.ErrTasks, true, func(rank int8) error {
		if trace != nil {
			trace(rank)
		}

		return nil
	})
}

func (r *Runner) runTask(updTicker *time.Ticker, task *model.Task, taskInfo string,
	isErrTasks bool,
) *Error {
	runner, ok := model.ValidTasks[task.Type]
	if !ok {
		return newError(types.TeExternalOperation, "unknown task type: %s", task.Type)
	}

	args, setupErr := r.setup(task)
	if setupErr != nil {
		r.logger.Error("%s: setup failed: %v", taskInfo, setupErr)

		return newErrorWith(types.TeExternalOperation,
			fmt.Sprintf("%s: setup failed", taskInfo), setupErr)
	}

	if validatorDB, ok := runner.(model.TaskValidatorDB); ok {
		if valErr := validatorDB.ValidateDB(r.db, args); valErr != nil {
			r.logger.Error("%s: validation failed %v", taskInfo, valErr)

			return newErrorWith(types.TeExternalOperation,
				fmt.Sprintf("%s: validation failed", taskInfo), valErr)
		}
	} else if validator, ok := runner.(model.TaskValidator); ok {
		if valErr := validator.Validate(args); valErr != nil {
			r.logger.Error("%s: validation failed %v", taskInfo, valErr)

			return newErrorWith(types.TeExternalOperation,
				fmt.Sprintf("%s: validation failed", taskInfo), valErr)
		}
	}

	r.logger.Debug("Executing task %s with args %s", taskInfo, mapToStr(args))

	var runErr error

	if isErrTasks {
		runErr = runner.Run(context.Background(), args, r.db, r.logger, r.transCtx)
	} else {
		runErr = runner.Run(r.Ctx, args, r.db, r.logger, r.transCtx)
	}

	if runErr != nil {
		var warningError *WarningError
		if !errors.As(runErr, &warningError) {
			r.logger.Error("%s: %v", taskInfo, runErr)
			r.transCtx.Transfer.ErrCode = types.TeExternalOperation
			r.transCtx.Transfer.ErrDetails = fmt.Sprintf("%s: %v", taskInfo, runErr)

			return newErrorWith(types.TeExternalOperation, taskInfo, runErr)
		}

		r.logger.Warning("%s: %v", taskInfo, runErr)
		r.transCtx.Transfer.ErrCode = types.TeWarning
		r.transCtx.Transfer.ErrDetails = fmt.Sprintf("%s: %v", taskInfo, runErr)
	} else {
		r.logger.Debug(taskInfo)
	}

	return r.updateProgress(updTicker, isErrTasks)
}

func (r *Runner) updateProgress(updTicker *time.Ticker, isErrTasks bool) *Error {
	r.transCtx.Transfer.TaskNumber++

	size := r.getFilesize()
	if size >= 0 && size != r.transCtx.Transfer.Filesize {
		r.transCtx.Transfer.Filesize = size
	}

	select {
	case <-updTicker.C:
	default:
		return nil
	}

	if dbErr := r.db.Update(r.transCtx.Transfer).Cols("task_number", "error_code",
		"error_details", "filesize").Run(); dbErr != nil {
		r.logger.Error("Failed to update transfer after task: %s", dbErr)

		if !isErrTasks {
			return newErrorWith(types.TeInternal, "failed to update transfer", dbErr)
		}
	}

	return nil
}

// runTasks executes sequentially the list of tasks given according to the
// Runner context.
func (r *Runner) runTasks(tasks []*model.Task, isErrTasks bool, trace func(rank int8) error,
) *Error {
	r.lock.Add(1)
	defer r.lock.Done()

	for i := r.transCtx.Transfer.TaskNumber; i < int8(len(tasks)); i++ {
		task := tasks[i]
		taskInfo := fmt.Sprintf("Task %s @ %s %s[%v]", task.Type, r.transCtx.Rule.Name,
			task.Chain, task.Rank)

		if !isErrTasks {
			select {
			case <-r.Ctx.Done():
				return ErrTransferInterrupted
			default:
			}
		}

		updTicker := time.NewTicker(time.Second)
		if err := r.runTask(updTicker, task, taskInfo, isErrTasks); err != nil {
			return err
		}

		if trace != nil {
			if err := trace(i); err != nil {
				return newErrorWith(types.TeInternal, "task trace error", err)
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

	info, err := fs.Stat(r.transCtx.Transfer.LocalPath)
	if err != nil {
		r.logger.Warning("Failed to retrieve file size: %s", err)

		return -1
	}

	return info.Size()
}
