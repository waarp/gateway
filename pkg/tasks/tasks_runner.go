// Package tasks regroups all the different types of transfer tasks runners.
package tasks

import (
	"bytes"
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
	Db       *database.Db
	Logger   *log.Logger
	Rule     *model.Rule
	Transfer *model.Transfer
	Signals  <-chan model.Signal
}

// GetTasks returns the list of all tasks of the given rule & chain.
func (p *Processor) GetTasks(chain model.Chain) ([]model.Task, *model.PipelineError) {
	list := []model.Task{}
	filters := &database.Filters{
		Order:      "rank ASC",
		Conditions: builder.And(builder.Eq{"rule_id": p.Rule.ID}, builder.Eq{"chain": chain}),
	}

	if err := p.Db.Select(&list, filters); err != nil {
		return nil, &model.PipelineError{Kind: model.KindDatabase}
	}
	return list, nil
}

func (p *Processor) runTask(task model.Task, taskInfo string) *model.PipelineError {
	runnable, ok := RunnableTasks[task.Type]
	if !ok {
		logMsg := fmt.Sprintf("%s: %s", taskInfo, "unknown task")
		return model.NewPipelineError(model.TeExternalOperation, logMsg)
	}
	args, err := p.setup(&task)
	if err != nil {
		logMsg := fmt.Sprintf("%s: %s", taskInfo, err.Error())
		p.Logger.Error(logMsg)
		return model.NewPipelineError(model.TeExternalOperation, logMsg)
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
		if err := p.Transfer.Update(p.Db); err != nil {
			p.Logger.Warningf("failed to update task status: %s", err.Error())
			return &model.PipelineError{Kind: model.KindDatabase}
		}
	}
	p.Logger.Info(logMsg)
	return nil
}

// RunTasks execute sequentially the list of tasks given
// according to the Processor context
func (p *Processor) RunTasks(tasks []model.Task) *model.PipelineError {
	for _, task := range tasks {
		taskInfo := fmt.Sprintf("Task %s @ %s %s[%v]", task.Type, p.Rule.Name,
			task.Chain, task.Rank)
		select {
		case signal := <-p.Signals:
			switch signal {
			case model.SignalShutdown:
				return &model.PipelineError{Kind: model.KindInterrupt}
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
	return nil
}

// setup contextualise and unmarshal the tasks arguments
// It return a json object exploitable by the task
func (p *Processor) setup(t *model.Task) (map[string]interface{}, error) {
	sArgs, err := p.replace(t)
	if err != nil {
		return nil, err
	}
	args := map[string]interface{}{}
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
			rep, err := f(p)
			if err != nil {
				return []byte(""), err
			}
			res = bytes.ReplaceAll(res, []byte(key), rep)
		}
	}
	return res, nil
}

type replacer func(*Processor) ([]byte, error)

var replacers = map[string]replacer{
	"#TRUEFULLPATH#": func(p *Processor) ([]byte, error) {
		if p.Rule.IsSend {
			return []byte(p.Transfer.SourcePath), nil
		}
		return []byte(p.Transfer.DestPath), nil
	},
	"#TRUEFILENAME#": func(p *Processor) ([]byte, error) {
		if p.Rule.IsSend {
			return []byte(filepath.Base(p.Transfer.SourcePath)), nil
		}
		return []byte(filepath.Base(p.Transfer.DestPath)), nil
	},
	"#ORIGINALFULLPATH#": func(p *Processor) ([]byte, error) {
		if p.Rule.IsSend {
			return []byte(p.Transfer.SourcePath), nil
		}
		return []byte(p.Transfer.DestPath), nil
	},
	"#ORIGINALFILENAME#": func(p *Processor) ([]byte, error) {
		if p.Rule.IsSend {
			return []byte(filepath.Base(p.Transfer.SourcePath)), nil
		}
		return []byte(filepath.Base(p.Transfer.DestPath)), nil
	},
	"#FILESIZE#": func(p *Processor) ([]byte, error) {
		return []byte("0"), nil
	},
	"#INPATH#": func(p *Processor) ([]byte, error) {
		if !p.Rule.IsSend {
			return []byte(p.Rule.Path), nil
		}
		return []byte{}, fmt.Errorf("send rule cannot use #INPATH#")
	},
	"#OUTPATH#": func(p *Processor) ([]byte, error) {
		if p.Rule.IsSend {
			return []byte(p.Rule.Path), nil
		}
		return []byte{}, fmt.Errorf("receive rule cannot use #OUTPATH#")
	},
	"#WORKPATH#": func(p *Processor) ([]byte, error) {
		// DEPRECATED
		return []byte{}, nil
	},
	"#ARCHPATH#": func(p *Processor) ([]byte, error) {
		// DEPRECATED
		return []byte{}, nil
	},
	"#HOMEPATH#": func(p *Processor) ([]byte, error) {
		// TODO ???
		return []byte{}, nil
	},
	"#RULE#": func(p *Processor) ([]byte, error) {
		return []byte(p.Rule.Name), nil
	},
	"#DATE#": func(p *Processor) ([]byte, error) {
		t := time.Now()
		return []byte(t.Format("20060102")), nil
	},
	"#HOUR#": func(p *Processor) ([]byte, error) {
		t := time.Now()
		return []byte(t.Format("030405")), nil
	},
	"#REMOTEHOST#": func(p *Processor) ([]byte, error) {
		if p.Transfer.IsServer {
			account := &model.LocalAccount{
				ID: p.Transfer.AccountID,
			}
			if err := p.Db.Get(account); err != nil {
				return []byte{}, err
			}
			return []byte(account.Login), nil
		}
		agent := &model.RemoteAgent{
			ID: p.Transfer.AgentID,
		}
		if err := p.Db.Get(agent); err != nil {
			return []byte{}, err
		}
		return []byte(agent.Name), nil
	},
	"#REMOTEHOSTIP#": func(p *Processor) ([]byte, error) {
		// TODO
		return []byte{}, nil
	},
	"#LOCALHOST#": func(p *Processor) ([]byte, error) {
		if p.Transfer.IsServer {
			agent := &model.LocalAgent{
				ID: p.Transfer.AgentID,
			}
			if err := p.Db.Get(agent); err != nil {
				return []byte{}, err
			}
			return []byte(agent.Name), nil
		}
		account := &model.RemoteAccount{
			ID: p.Transfer.AccountID,
		}
		if err := p.Db.Get(account); err != nil {
			return []byte{}, err
		}
		return []byte(account.Login), nil
	},
	"#LOCALHOSTIP#": func(p *Processor) ([]byte, error) {
		// TODO
		return []byte{}, nil
	},
	"#TRANFERID#": func(p *Processor) ([]byte, error) {
		return []byte(fmt.Sprint(p.Transfer.ID)), nil
	},
	"#REQUESTERHOST#": func(p *Processor) ([]byte, error) {
		client, err := getClient(p)
		return []byte(client), err
	},
	"#REQUESTEDHOST#": func(p *Processor) ([]byte, error) {
		server, err := getServer(p)
		return []byte(server), err
	},
	"#FULLTRANFERID#": func(p *Processor) ([]byte, error) {
		//DEPRECATED
		client, err := getClient(p)
		if err != nil {
			return []byte{}, nil
		}
		server, err := getServer(p)
		if err != nil {
			return []byte{}, nil
		}
		return []byte(fmt.Sprintf("%d_%s_%s", p.Transfer.ID, client, server)), nil
	},
	"#RANKTRANSFER#": func(p *Processor) ([]byte, error) {
		return []byte("0"), nil
	},
	"#BLOCKSIZE#": func(p *Processor) ([]byte, error) {
		return []byte("1"), nil
	},
	"#ERRORMSG#": func(p *Processor) ([]byte, error) {
		return []byte(p.Transfer.Error.Details), nil
	},
	"#ERRORCODE#": func(p *Processor) ([]byte, error) {
		return []byte{p.Transfer.Error.Code.R66Code()}, nil
	},
	"#ERRORSTRCODE#": func(p *Processor) ([]byte, error) {
		return []byte(p.Transfer.Error.Details), nil
	},
	"#NOWAIT#": func(p *Processor) ([]byte, error) {
		return []byte{}, nil
	},
	"#LOCALEXEC#": func(p *Processor) ([]byte, error) {
		return []byte{}, nil
	},
}

func getClient(p *Processor) (string, error) {
	if p.Transfer.IsServer {
		account := &model.LocalAccount{
			ID: p.Transfer.AccountID,
		}
		if err := p.Db.Get(account); err != nil {
			return "", err
		}
		return account.Login, nil
	}
	account := &model.RemoteAccount{
		ID: p.Transfer.AccountID,
	}
	if err := p.Db.Get(account); err != nil {
		return "", err
	}
	return account.Login, nil
}

func getServer(p *Processor) (string, error) {
	if p.Transfer.IsServer {
		agent := &model.LocalAgent{
			ID: p.Transfer.AgentID,
		}
		if err := p.Db.Get(agent); err != nil {
			return "", err
		}
		return agent.Name, nil
	}
	agent := &model.RemoteAgent{
		ID: p.Transfer.AgentID,
	}
	if err := p.Db.Get(agent); err != nil {
		return "", err
	}
	return agent.Name, nil
}
