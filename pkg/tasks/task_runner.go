// Package tasks regroups all the different types of transfer tasks runners.
package tasks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
)

// Runner provides a way to execute tasks
// given a transfer context (rule, transfer)
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
func (r *Runner) PreTasks() error {
	return r.runTasks(r.transCtx.PreTasks, false)
}

// PostTasks executes the transfer's post-tasks.
func (r *Runner) PostTasks() error {
	return r.runTasks(r.transCtx.PostTasks, false)
}

// ErrorTasks executes the transfer's error-tasks.
func (r *Runner) ErrorTasks() error {
	return r.runTasks(r.transCtx.ErrTasks, true)
}

func (r *Runner) runTask(task model.Task, taskInfo string, isErrTasks bool) error {
	runner, ok := model.ValidTasks[task.Type]
	if !ok {
		logMsg := fmt.Sprintf("%s: unknown task", taskInfo)
		return types.NewTransferError(types.TeExternalOperation, logMsg)
	}
	args, err := r.setup(&task)
	if err != nil {
		logMsg := fmt.Sprintf("%s: %s", taskInfo, err.Error())
		r.logger.Error(logMsg)
		return types.NewTransferError(types.TeExternalOperation, logMsg)
	}

	if err := runner.Validate(args); err != nil {
		logMsg := fmt.Sprintf("%s: %s", taskInfo, err.Error())
		r.logger.Error(logMsg)
		return types.NewTransferError(types.TeExternalOperation, logMsg)
	}

	var msg string
	if isErrTasks {
		msg, err = runner.Run(args, r.db, r.transCtx, context.Background())
	} else {
		msg, err = runner.Run(args, r.db, r.transCtx, r.ctx)
	}

	if err != nil {
		errMsg := fmt.Sprintf("%s: %s", taskInfo, err)
		if _, ok := err.(*errWarning); !ok {
			r.logger.Error(errMsg)
			r.transCtx.Transfer.Error = types.NewTransferError(types.TeExternalOperation, errMsg)
			return types.NewTransferError(types.TeExternalOperation, errMsg)
		}

		r.logger.Warning(errMsg)
		r.transCtx.Transfer.Error = types.NewTransferError(types.TeWarning, errMsg)
		if dbErr := r.db.Update(r.transCtx.Transfer).Cols("error_code", "error_details").Run(); dbErr != nil {
			r.logger.Errorf("Failed to update task status: %s", dbErr)
			if !isErrTasks {
				return types.NewTransferError(types.TeInternal, dbErr.Error())
			}
		}
	} else if msg != "" {
		r.logger.Debugf("%s: %s", taskInfo, msg)
	} else {
		r.logger.Debug(taskInfo)
	}

	r.transCtx.Transfer.TaskNumber++
	if err := r.db.Update(r.transCtx.Transfer).Cols("task_number").Run(); err != nil {
		r.logger.Errorf("Failed to update task number: %s", err.Error())
		if !isErrTasks {
			return types.NewTransferError(types.TeInternal, err.Error())
		}
	}
	return nil
}

// runTasks execute sequentially the list of tasks given
// according to the Runner context
func (r *Runner) runTasks(tasks []model.Task, isErrTasks bool) error {
	r.lock.Add(1)
	defer r.lock.Done()

	for i := r.transCtx.Transfer.TaskNumber; i < uint64(len(tasks)); i++ {
		task := tasks[i]
		taskInfo := fmt.Sprintf("Task %s @ %s %s[%v]", task.Type, r.transCtx.Rule.Name,
			task.Chain, task.Rank)
		if !isErrTasks {
			select {
			case <-r.ctx.Done():
				return r.ctx.Err()
			default:
			}
		}

		if err := r.runTask(task, taskInfo, isErrTasks); err != nil {
			return err
		}
	}

	r.transCtx.Transfer.TaskNumber = 0
	if err := r.db.Update(r.transCtx.Transfer).Cols("task_number").Run(); err != nil {
		r.logger.Errorf("Failed to reset task number: %s", err.Error())
		if !isErrTasks {
			return types.NewTransferError(types.TeInternal, err.Error())
		}
	}

	return nil
}

// setup contextualise and unmarshal the tasks arguments
// It return a json object exploitable by the task
func (r *Runner) setup(t *model.Task) (map[string]string, error) {
	sArgs, err := r.replace(t)
	if err != nil {
		return nil, err
	}
	args := map[string]string{}
	if err := json.Unmarshal(sArgs, &args); err != nil {
		return nil, err
	}
	return args, nil
}

// replace replace all the context variables (#varname#) in the tasks arguments
// by their context value
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
				return nil, err
			}
			rep = rep[1 : len(rep)-1]

			res = bytes.ReplaceAll(res, []byte(key), rep)
		}
	}
	return res, nil
}
