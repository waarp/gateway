package tasks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

// TasksRunner provides a way to execute tasks
// given a transfer context (rule, transfer)
type TasksRunner struct {
	Rule     *model.Rule
	Transfer *model.Transfer
}

// RunTasks execute sequentialy the list of tasks given
// according to the TasksRunner context
func (r *TasksRunner) RunTasks(tasks []*model.Task) error {
	for _, task := range tasks {
		runnable, ok := RunnableTasks[task.Type]
		if !ok {
			return fmt.Errorf("unknown task")
		}
		args, err := r.setup(task)
		if err != nil {
			return err
		}
		err = runnable.Run(args, r.Rule, r.Transfer)
		if err != nil {
			return err
		}
	}
	return nil
}

// setup contextualise and unmarshal the tasks arguments
// It return a json object exploitable by the task
func (r *TasksRunner) setup(t *model.Task) (map[string]interface{}, error) {
	sArgs := r.replace(t)
	args := map[string]interface{}{}
	if err := json.Unmarshal(sArgs, &args); err != nil {
		return nil, err
	}
	return args, nil
}

// replace replace all the context variables (#varname#) in the tasks arguments
// by their context value
func (r *TasksRunner) replace(t *model.Task) []byte {
	res := t.Args
	for key, f := range replacers {
		if bytes.Contains(res, []byte(key)) {
			res = bytes.ReplaceAll(res, []byte(key), f(r))
		}
	}
	return res
}

type replacer func(*TasksRunner) []byte

var replacers = map[string]replacer{
	"#TRUEFULLPATH#": func(r *TasksRunner) []byte {
		if r.Rule.IsSend {
			return []byte(r.Transfer.SourcePath)
		}
		return []byte(r.Transfer.DestPath)
	},
	"#TRUEFILENAME#": func(r *TasksRunner) []byte {
		if r.Rule.IsSend {
			return []byte(filepath.Base(r.Transfer.SourcePath))
		}
		return []byte(filepath.Base(r.Transfer.DestPath))
	},
	"#ORIGINALFULLPATH#": func(r *TasksRunner) []byte {
		if r.Rule.IsSend {
			return []byte(r.Transfer.SourcePath)
		}
		return []byte(r.Transfer.DestPath)
	},
	"#ORIGINALFILENAME#": func(r *TasksRunner) []byte {
		if r.Rule.IsSend {
			return []byte(filepath.Base(r.Transfer.SourcePath))
		}
		return []byte(filepath.Base(r.Transfer.DestPath))
	},
	"#FILESIZE#": func(r *TasksRunner) []byte {
		return []byte("0")
	},
	"#INPATH#": func(r *TasksRunner) []byte {
		if !r.Rule.IsSend {
			return []byte(r.Rule.Path)
		}
		return []byte("")
	},
	"#OUTPATH#": func(r *TasksRunner) []byte {
		if r.Rule.IsSend {
			return []byte(r.Rule.Path)
		}
		return []byte("")
	},
	"#WORKPATH#": func(r *TasksRunner) []byte {
		// DEPRECATED
		return []byte("")
	},
	"#ARCHPATH#": func(r *TasksRunner) []byte {
		// DEPRECATED
		return []byte("")
	},
	"#HOMEPATH#": func(r *TasksRunner) []byte {
		// TODO ???
		return []byte("")
	},
	"#RULE#": func(r *TasksRunner) []byte {
		return []byte(r.Rule.Name)
	},
	"#DATE#": func(r *TasksRunner) []byte {
		t := time.Now()
		return []byte(t.Format("20060102"))
	},
	"#HOUR#": func(r *TasksRunner) []byte {
		t := time.Now()
		return []byte(t.Format("030405"))
	},
	"#REMOTEHOST#": func(r *TasksRunner) []byte {
		// TODO
		return []byte("")
	},
	"#LOCALHOST#": func(r *TasksRunner) []byte {
		// TODO
		return []byte("")
	},
	"#REMOTEHOSTIP#": func(r *TasksRunner) []byte {
		// TODO
		return []byte("")
	},
	"#LOCALHOSTIP#": func(r *TasksRunner) []byte {
		// TODO
		return []byte("")
	},
	"#REQUESTERHOST#": func(r *TasksRunner) []byte {
		// TODO
		return []byte("")
	},
	"#REQUESTEDHOST#": func(r *TasksRunner) []byte {
		// TODO
		return []byte("")
	},
}
