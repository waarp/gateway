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

var (
	// ErrShutdown is the error returned when a shutdown signal is received
	// during the execution of tasks.
	ErrShutdown = errors.New("server shutdown signal received")

	errWarning = errors.New("warning")
)

// GetTasks returns the list of all tasks of the given rule & chain.
func GetTasks(db *database.Db, ruleID uint64, chain model.Chain) ([]*model.Task, error) {
	list := []*model.Task{}
	filters := &database.Filters{
		Order:      "rank ASC",
		Conditions: builder.Eq{"rule_id": ruleID, "chain": chain},
	}

	if err := db.Select(&list, filters); err != nil {
		return nil, err
	}
	return list, nil
}

// Processor provides a way to execute tasks
// given a transfer context (rule, transfer)
type Processor struct {
	Db       *database.Db
	Logger   *log.Logger
	Rule     *model.Rule
	Transfer *model.Transfer
	Shutdown <-chan bool
}

// RunTasks execute sequentially the list of tasks given
// according to the Processor context
func (p *Processor) RunTasks(tasks []*model.Task) error {
	for _, task := range tasks {
		taskInfo := fmt.Sprintf("Task %s @ %s %s[%v]", task.Type, p.Rule.Name,
			task.Chain, task.Rank)
		select {
		case <-p.Shutdown:
			return ErrShutdown
		default:
		}

		taskErr := func() error {
			runnable, ok := RunnableTasks[task.Type]
			if !ok {
				return fmt.Errorf("unknown task")
			}
			args, err := p.setup(task)
			if err != nil {
				return err
			}

			msg, err := runnable.Run(args, p)
			logMsg := fmt.Sprintf("%s: %s", taskInfo, msg)
			if err != nil {
				if err != errWarning {
					return err
				}
				trans := &model.Transfer{
					Error: model.NewTransferError(model.TeWarning, logMsg),
				}
				if err := p.Db.Update(trans, p.Transfer.ID, false); err != nil {
					return err
				}
				p.Transfer.Error = trans.Error
				p.Logger.Warning(logMsg)
			} else {
				p.Logger.Info(logMsg)
			}
			return nil
		}()
		if taskErr != nil {
			logMsg := fmt.Sprintf("%s: %s", taskInfo, taskErr.Error())
			trans := &model.Transfer{
				Error: model.NewTransferError(model.TeExternalOperation, logMsg),
			}
			if err := p.Db.Update(trans, p.Transfer.ID, false); err != nil {
				return err
			}
			p.Transfer.Error = trans.Error
			p.Logger.Error(logMsg)
			return taskErr
		}
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
			return p.Transfer.SourcePath, nil
		}
		return p.Transfer.DestPath, nil
	},
	"#TRUEFILENAME#": func(p *Processor) (string, error) {
		if p.Rule.IsSend {
			return filepath.Base(p.Transfer.SourcePath), nil
		}
		return filepath.Base(p.Transfer.DestPath), nil
	},
	"#ORIGINALFULLPATH#": func(p *Processor) (string, error) {
		if p.Rule.IsSend {
			return p.Transfer.SourcePath, nil
		}
		return p.Transfer.DestPath, nil
	},
	"#ORIGINALFILENAME#": func(p *Processor) (string, error) {
		if p.Rule.IsSend {
			return filepath.Base(p.Transfer.SourcePath), nil
		}
		return filepath.Base(p.Transfer.DestPath), nil
	},
	"#FILESIZE#": func(p *Processor) (string, error) {
		return "0", nil
	},
	"#INPATH#": func(p *Processor) (string, error) {
		if !p.Rule.IsSend {
			return p.Rule.Path, nil
		}
		return "", fmt.Errorf("send rule cannot use #INPATH#")
	},
	"#OUTPATH#": func(p *Processor) (string, error) {
		if p.Rule.IsSend {
			return p.Rule.Path, nil
		}
		return "", fmt.Errorf("receive rule cannot use #OUTPATH#")
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
			if err := p.Db.Get(account); err != nil {
				return "", err
			}
			return account.Login, nil
		}
		agent := &model.RemoteAgent{
			ID: p.Transfer.AgentID,
		}
		if err := p.Db.Get(agent); err != nil {
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
			if err := p.Db.Get(agent); err != nil {
				return "", err
			}
			return agent.Name, nil
		}
		account := &model.RemoteAccount{
			ID: p.Transfer.AccountID,
		}
		if err := p.Db.Get(account); err != nil {
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
