package tasks

import (
	"context"
	"fmt"
	"path"
	"strings"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
)

// execOutputTask is a task which executes an external program which moves the
// transferred file.
type execOutputTask struct{}

func init() {
	model.ValidTasks["EXECOUTPUT"] = &execOutputTask{}
}

// Validate checks if the EXECMOVE task has all the required arguments.
func (e *execOutputTask) Validate(params map[string]string) error {

	if _, _, _, err := parseExecArgs(params); err != nil {
		return fmt.Errorf("failed to parse task arguments: %s", err.Error())
	}

	return nil
}

func getNewFileName(output string) string {
	lines := strings.Split(strings.TrimSpace(output), lineSeparator)
	lastLine := lines[len(lines)-1]

	if strings.HasPrefix(lastLine, "NEWFILENAME:") {
		return strings.TrimPrefix(lastLine, "NEWFILENAME:")
	}
	return ""
}

// Run executes the task by executing an external program with the given parameters.
func (e *execOutputTask) Run(parent context.Context, params map[string]string,
	_ *database.DB, transCtx *model.TransferContext) (string, error) {

	output, cmdErr := runExec(parent, params, true)
	msg := ""
	if output != nil {
		msg = output.String()
	}

	if cmdErr != nil {
		if newPath := getNewFileName(msg); newPath != "" {
			transCtx.Transfer.LocalPath = utils.ToOSPath(newPath)
			transCtx.Transfer.RemotePath = utils.ToStandardPath(path.Dir(
				transCtx.Transfer.RemotePath), path.Base(transCtx.Transfer.LocalPath))
		}

		return "", cmdErr
	}

	return msg, nil
}
