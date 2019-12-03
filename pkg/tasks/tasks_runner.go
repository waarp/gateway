// Package tasks regroups all the different types of transfer tasks runners.
package tasks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
)

var errWarning = fmt.Errorf("warning")

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
}

// RunTasks execute sequentially the list of tasks given
// according to the Processor context
func (p *Processor) RunTasks(tasks []*model.Task) error {
	for _, task := range tasks {
		taskInfo := fmt.Sprintf("Task %s @ %s %s[%v]", task.Type, p.Rule.Name,
			task.Chain, task.Rank)

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
func (p *Processor) setup(t *model.Task) (map[string]interface{}, error) {
	sArgs := p.replace(t)
	args := map[string]interface{}{}
	if err := json.Unmarshal(sArgs, &args); err != nil {
		return nil, err
	}
	return args, nil
}

// replace replace all the context variables (#varname#) in the tasks arguments
// by their context value
func (p *Processor) replace(t *model.Task) []byte {
	res := t.Args
	for key, f := range replacers {
		if bytes.Contains(res, []byte(key)) {
			res = bytes.ReplaceAll(res, []byte(key), f(p))
		}
	}
	return res
}

type replacer func(*Processor) []byte

var replacers = map[string]replacer{
	"#TRUEFULLPATH#": func(r *Processor) []byte {
		if r.Rule.IsSend {
			return []byte(r.Transfer.SourcePath)
		}
		return []byte(r.Transfer.DestPath)
	},
	"#TRUEFILENAME#": func(r *Processor) []byte {
		if r.Rule.IsSend {
			return []byte(filepath.Base(r.Transfer.SourcePath))
		}
		return []byte(filepath.Base(r.Transfer.DestPath))
	},
	"#ORIGINALFULLPATH#": func(r *Processor) []byte {
		if r.Rule.IsSend {
			return []byte(r.Transfer.SourcePath)
		}
		return []byte(r.Transfer.DestPath)
	},
	"#ORIGINALFILENAME#": func(r *Processor) []byte {
		if r.Rule.IsSend {
			return []byte(filepath.Base(r.Transfer.SourcePath))
		}
		return []byte(filepath.Base(r.Transfer.DestPath))
	},
	"#FILESIZE#": func(r *Processor) []byte {
		return []byte("0")
	},
	"#INPATH#": func(r *Processor) []byte {
		if !r.Rule.IsSend {
			return []byte(r.Rule.Path)
		}
		return []byte("")
	},
	"#OUTPATH#": func(r *Processor) []byte {
		if r.Rule.IsSend {
			return []byte(r.Rule.Path)
		}
		return []byte("")
	},
	"#WORKPATH#": func(r *Processor) []byte {
		// DEPRECATED
		return []byte("")
	},
	"#ARCHPATH#": func(r *Processor) []byte {
		// DEPRECATED
		return []byte("")
	},
	"#HOMEPATH#": func(r *Processor) []byte {
		// TODO ???
		return []byte("")
	},
	"#RULE#": func(r *Processor) []byte {
		return []byte(r.Rule.Name)
	},
	"#DATE#": func(r *Processor) []byte {
		t := time.Now()
		return []byte(t.Format("20060102"))
	},
	"#HOUR#": func(r *Processor) []byte {
		t := time.Now()
		return []byte(t.Format("030405"))
	},
	"#REMOTEHOST#": func(r *Processor) []byte {
		// TODO
		return []byte("")
	},
	"#LOCALHOST#": func(r *Processor) []byte {
		// TODO
		return []byte("")
	},
	"#REMOTEHOSTIP#": func(r *Processor) []byte {
		// TODO
		return []byte("")
	},
	"#LOCALHOSTIP#": func(r *Processor) []byte {
		// TODO
		return []byte("")
	},
	"#REQUESTERHOST#": func(r *Processor) []byte {
		// TODO
		return []byte("")
	},
	"#REQUESTEDHOST#": func(r *Processor) []byte {
		// TODO
		return []byte("")
	},
}
