package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

// ExecOutputTask is a task which executes an external program which moves the
// transferred file.
type ExecOutputTask struct{}

func init() {
	RunnableTasks["EXECOUTPUT"] = &ExecOutputTask{}
}

// Validate checks if the EXECMOVE task has all the required arguments.
func (e *ExecOutputTask) Validate(task *model.Task) error {
	var args map[string]interface{}
	if err := json.Unmarshal(task.Args, &args); err != nil {
		return err
	}

	if path, ok := args["path"].(string); !ok || path == "" {
		return fmt.Errorf("missing program path")
	}
	if _, ok := args["args"].(string); !ok {
		return fmt.Errorf("missing program arguments")
	}
	if delay, ok := args["delay"].(float64); !ok || delay == 0 {
		return fmt.Errorf("missing program delay")
	}

	return nil
}

func getNewFileName(output string) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	lastLine := lines[len(lines)-1]

	if strings.HasPrefix(lastLine, "NEWFILENAME:") {
		return strings.TrimPrefix(lastLine, "NEWFILENAME:")
	}
	return ""
}

// Run executes the task by executing an external program with the given parameters.
func (e *ExecOutputTask) Run(params map[string]interface{}, processor *Processor) (string, error) {
	var path, args string
	var delay float64
	var ok bool
	if path, ok = params["path"].(string); !ok || path == "" {
		return "missing program path", fmt.Errorf("missing program path")
	}
	if args, ok = params["args"].(string); !ok {
		return "missing program arguments", fmt.Errorf("missing program arguments")
	}
	if delay, ok = params["delay"].(float64); !ok || delay == 0 {
		return "missing program delay", fmt.Errorf("missing program delay")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(delay)*
		time.Millisecond)
	defer cancel()

	cmd := getCommand(ctx, path, args)
	in, out, err := os.Pipe()
	if err != nil {
		return err.Error(), err
	}
	defer func() {
		_ = in.Close()
		_ = out.Close()
	}()
	cmd.Stdout = out

	cmdErr := cmd.Run()
	if cmdErr == nil {
		return "", nil
	}

	select {
	case <-ctx.Done():
		return "max exec delay expired", fmt.Errorf("max exec delay expired")
	default:
		if ex, ok := cmdErr.(*exec.ExitError); ok && ex.ExitCode() == 1 {
			return cmdErr.Error(), errWarning
		}
	}

	_ = out.Close()
	msg, err := ioutil.ReadAll(in)
	if err != nil {
		return err.Error(), err
	}

	if newPath := getNewFileName(string(msg)); newPath != "" {
		if processor.Rule.IsSend {
			processor.Transfer.SourcePath = newPath
		} else {
			processor.Transfer.DestPath = newPath
		}
	}
	return string(msg), cmdErr
}
