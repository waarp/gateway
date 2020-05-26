// Package tasks regroups all the different types of transfer tasks runners.
package tasks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
)

var errWarning = errors.New("warning")

// Processor provides a way to execute tasks
// given a transfer context (rule, transfer)
type Processor struct {
	DB       *database.DB
	Logger   *log.Logger
	Rule     *model.Rule
	Transfer *model.Transfer
	Signals  <-chan model.Signal
	Ctx      context.Context
	InPath   string
	OutPath  string
}

// GetTasks returns the list of all tasks of the given rule & chain.
func (p *Processor) GetTasks(chain model.Chain) ([]model.Task, *model.PipelineError) {
	list := []model.Task{}
	filters := &database.Filters{
		Order:      "rank ASC",
		Conditions: builder.And(builder.Eq{"rule_id": p.Rule.ID}, builder.Eq{"chain": chain}),
	}

	if err := p.DB.Select(&list, filters); err != nil {
		return nil, &model.PipelineError{Kind: model.KindDatabase}
	}
	return list, nil
}

func (p *Processor) runTask(task model.Task, taskInfo string) *model.PipelineError {
	runnable, ok := RunnableTasks[task.Type]
	if !ok {
		logMsg := fmt.Sprintf("%s: unknown task", taskInfo)
		return model.NewPipelineError(model.TeExternalOperation, logMsg)
	}
	args, err := p.setup(&task)
	if err != nil {
		logMsg := fmt.Sprintf("%s: %s", taskInfo, err.Error())
		p.Logger.Error(logMsg)
		return model.NewPipelineError(model.TeExternalOperation, logMsg)
	}

	if val, ok := runnable.(model.Validator); ok {
		if err := val.Validate(args); err != nil {
			logMsg := fmt.Sprintf("%s: %s", taskInfo, err.Error())
			p.Logger.Error(logMsg)
			return model.NewPipelineError(model.TeExternalOperation, logMsg)
		}
	}

	msg, err := runnable.Run(args, p)
	logMsg := fmt.Sprintf("%s: %s", taskInfo, msg)
	if err != nil {
		if err != errWarning {
			p.Logger.Error(logMsg)
			return model.NewPipelineError(model.TeExternalOperation, logMsg)
		}
		p.Logger.Warning(logMsg)
		p.Transfer.Error = model.NewTransferError(model.TeWarning, logMsg)
		if err := p.Transfer.Update(p.DB); err != nil {
			p.Logger.Warningf("failed to update task status: %s", err.Error())
			return &model.PipelineError{Kind: model.KindDatabase}
		}
	}
	p.Transfer.TaskNumber++
	if err := p.Transfer.Update(p.DB); err != nil {
		p.Logger.Warningf("failed to update task number: %s", err.Error())
		return &model.PipelineError{Kind: model.KindDatabase}
	}
	p.Logger.Info(logMsg)
	return nil
}

// RunTasks execute sequentially the list of tasks given
// according to the Processor context
func (p *Processor) RunTasks(tasks []model.Task) *model.PipelineError {
	for i := p.Transfer.TaskNumber; i < uint64(len(tasks)); i++ {
		task := tasks[i]
		taskInfo := fmt.Sprintf("Task %s @ %s %s[%v]", task.Type, p.Rule.Name,
			task.Chain, task.Rank)
		select {
		case <-p.Ctx.Done():
			return &model.PipelineError{Kind: model.KindInterrupt}
		case signal := <-p.Signals:
			switch signal {
			case model.SignalCancel:
				return &model.PipelineError{Kind: model.KindCancel}
			case model.SignalPause:
				return &model.PipelineError{Kind: model.KindPause}
			}
		default:
		}

		if err := p.runTask(task, taskInfo); err != nil {
			return err
		}
	}
	p.Transfer.TaskNumber = 0
	if err := p.Transfer.Update(p.DB); err != nil {
		p.Logger.Warningf("failed to update task number: %s", err.Error())
		return &model.PipelineError{Kind: model.KindDatabase}
	}
	return nil
}

// setup contextualise and unmarshal the tasks arguments
// It return a json object exploitable by the task
func (p *Processor) setup(t *model.Task) (map[string]string, error) {
	sArgs, err := p.replace(t)
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
func (p *Processor) replace(t *model.Task) ([]byte, error) {
	res := t.Args
	for key, f := range replacers {
		if bytes.Contains(res, []byte(key)) {
			r, err := f(p)
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

type replacer func(*Processor) (string, error)

var replacers = map[string]replacer{
	"#TRUEFULLPATH#": func(p *Processor) (string, error) {
		if p.Rule.IsSend {
			return p.Transfer.SourceFile, nil
		}
		return p.Transfer.DestFile, nil
	},
	"#TRUEFILENAME#": func(p *Processor) (string, error) {
		if p.Rule.IsSend {
			return filepath.Base(p.Transfer.SourceFile), nil
		}
		return filepath.Base(p.Transfer.DestFile), nil
	},
	"#ORIGINALFULLPATH#": func(p *Processor) (string, error) {
		if p.Rule.IsSend {
			return p.Transfer.SourceFile, nil
		}
		return p.Transfer.DestFile, nil
	},
	"#ORIGINALFILENAME#": func(p *Processor) (string, error) {
		if p.Rule.IsSend {
			return filepath.Base(p.Transfer.SourceFile), nil
		}
		return filepath.Base(p.Transfer.DestFile), nil
	},
	"#FILESIZE#": func(p *Processor) (string, error) {
		return "0", nil
	},
	"#INPATH#": func(p *Processor) (string, error) {
		return p.InPath, nil
	},
	"#OUTPATH#": func(p *Processor) (string, error) {
		return p.OutPath, nil
	},
	"#WORKPATH#": func(p *Processor) (string, error) {
		// DEPRECATED
		return "", nil
	},
	"#ARCHPATH#": func(p *Processor) (string, error) {
		// DEPRECATED
		return "", nil
	},
	"#HOMEPATH#": func(p *Processor) (string, error) {
		// TODO ???
		return "", nil
	},
	"#RULE#": func(p *Processor) (string, error) {
		return p.Rule.Name, nil
	},
	"#DATE#": func(p *Processor) (string, error) {
		t := time.Now()
		return t.Format("20060102"), nil
	},
	"#HOUR#": func(p *Processor) (string, error) {
		t := time.Now()
		return t.Format("030405"), nil
	},
	"#REMOTEHOST#": func(p *Processor) (string, error) {
		if p.Transfer.IsServer {
			account := &model.LocalAccount{
				ID: p.Transfer.AccountID,
			}
			if err := p.DB.Get(account); err != nil {
				return "", err
			}
			return account.Login, nil
		}
		agent := &model.RemoteAgent{
			ID: p.Transfer.AgentID,
		}
		if err := p.DB.Get(agent); err != nil {
			return "", err
		}
		return agent.Name, nil
	},
	"#REMOTEHOSTIP#": func(p *Processor) (string, error) {
		// TODO
		return "", nil
	},
	"#LOCALHOST#": func(p *Processor) (string, error) {
		if p.Transfer.IsServer {
			agent := &model.LocalAgent{
				ID: p.Transfer.AgentID,
			}
			if err := p.DB.Get(agent); err != nil {
				return "", err
			}
			return agent.Name, nil
		}
		account := &model.RemoteAccount{
			ID: p.Transfer.AccountID,
		}
		if err := p.DB.Get(account); err != nil {
			return "", err
		}
		return account.Login, nil
	},
	"#LOCALHOSTIP#": func(p *Processor) (string, error) {
		// TODO
		return "", nil
	},
	"#TRANFERID#": func(p *Processor) (string, error) {
		return fmt.Sprint(p.Transfer.ID), nil
	},
	"#REQUESTERHOST#": func(p *Processor) (string, error) {
		client, err := getClient(p)
		return client, err
	},
	"#REQUESTEDHOST#": func(p *Processor) (string, error) {
		server, err := getServer(p)
		return server, err
	},
	"#FULLTRANFERID#": func(p *Processor) (string, error) {
		//DEPRECATED
		client, err := getClient(p)
		if err != nil {
			return "", nil
		}
		server, err := getServer(p)
		if err != nil {
			return "", nil
		}
		return fmt.Sprintf("%d_%s_%s", p.Transfer.ID, client, server), nil
	},
	"#RANKTRANSFER#": func(p *Processor) (string, error) {
		return "0", nil
	},
	"#BLOCKSIZE#": func(p *Processor) (string, error) {
		return "1", nil
	},
	"#ERRORMSG#": func(p *Processor) (string, error) {
		return p.Transfer.Error.Details, nil
	},
	"#ERRORCODE#": func(p *Processor) (string, error) {
		return string(p.Transfer.Error.Code.R66Code()), nil
	},
	"#ERRORSTRCODE#": func(p *Processor) (string, error) {
		return p.Transfer.Error.Details, nil
	},
	"#NOWAIT#": func(p *Processor) (string, error) {
		return "", nil
	},
	"#LOCALEXEC#": func(p *Processor) (string, error) {
		return "", nil
	},
}

func getClient(p *Processor) (string, error) {
	if p.Transfer.IsServer {
		account := &model.LocalAccount{
			ID: p.Transfer.AccountID,
		}
		if err := p.DB.Get(account); err != nil {
			return "", err
		}
		return account.Login, nil
	}
	account := &model.RemoteAccount{
		ID: p.Transfer.AccountID,
	}
	if err := p.DB.Get(account); err != nil {
		return "", err
	}
	return account.Login, nil
}

func getServer(p *Processor) (string, error) {
	if p.Transfer.IsServer {
		agent := &model.LocalAgent{
			ID: p.Transfer.AgentID,
		}
		if err := p.DB.Get(agent); err != nil {
			return "", err
		}
		return agent.Name, nil
	}
	agent := &model.RemoteAgent{
		ID: p.Transfer.AgentID,
	}
	if err := p.DB.Get(agent); err != nil {
		return "", err
	}
	return agent.Name, nil
}
