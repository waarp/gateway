// Package tasks regroups all the different types of transfer tasks runners.
package tasks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
)

// Runner provides a way to execute tasks
// given a transfer context (rule, transfer)
type Runner struct {
	db     *database.DB
	logger *log.Logger
	info   *model.TransferContext
	ctx    context.Context
}

// NewTaskRunner returns a new tasks.Runner using the given elements.
func NewTaskRunner(db *database.DB, logger *log.Logger, info *model.TransferContext,
	ctx context.Context) *Runner {
	return &Runner{
		db:     db,
		logger: logger,
		info:   info,
		ctx:    ctx,
	}
}

// PreTasks executes the transfer's pre-tasks.
func (r *Runner) PreTasks() error {
	return r.runTasks(r.info.PreTasks)
}

// PostTasks executes the transfer's post-tasks.
func (r *Runner) PostTasks() error {
	return r.runTasks(r.info.PostTasks)
}

// ErrorTasks executes the transfer's error-tasks.
func (r *Runner) ErrorTasks() error {
	return r.runTasks(r.info.ErrTasks)
}

func (r *Runner) runTask(task model.Task, taskInfo string) error {
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

	msg, err := runner.Run(args, nil, r.info, nil)
	if err != nil {
		errMsg := fmt.Sprintf("%s: %s", taskInfo, err)
		if _, ok := err.(*errWarning); !ok {
			r.logger.Error(errMsg)
			r.info.Transfer.Error = types.NewTransferError(types.TeExternalOperation, errMsg)
			return types.NewTransferError(types.TeExternalOperation, fmt.Sprintf(
				"%s-task failed", strings.ToLower(string(task.Chain))))
		}

		r.logger.Warning(errMsg)
		r.info.Transfer.Error = types.NewTransferError(types.TeWarning, errMsg)
		if dbErr := r.db.Update(r.info.Transfer).Cols("error_code", "error_details").Run(); dbErr != nil {
			r.logger.Errorf("Failed to update task status: %s", dbErr)
			return dbErr
		}
	} else if msg != "" {
		r.logger.Debugf("%s: %s", taskInfo, msg)
	} else {
		r.logger.Debug(taskInfo)
	}

	r.info.Transfer.TaskNumber++
	if err := r.db.Update(r.info.Transfer).Cols("task_number").Run(); err != nil {
		r.logger.Warningf("Failed to update task number: %s", err.Error())
		return err
	}
	return nil
}

// runTasks execute sequentially the list of tasks given
// according to the Runner context
func (r *Runner) runTasks(tasks []model.Task) error {
	for i := r.info.Transfer.TaskNumber; i < uint64(len(tasks)); i++ {
		task := tasks[i]
		taskInfo := fmt.Sprintf("Task %s @ %s %s[%v]", task.Type, r.info.Rule.Name,
			task.Chain, task.Rank)
		select {
		case <-r.ctx.Done():
			return &model.ShutdownError{}
		default:
		}

		if err := r.runTask(task, taskInfo); err != nil {
			return err
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
