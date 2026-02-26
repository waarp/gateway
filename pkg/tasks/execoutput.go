package tasks

import (
	"bufio"
	"context"
	"fmt"
	"path"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// execOutputTask is a task which executes an external program which moves the
// transferred file.
type execOutputTask struct{}

// Validate checks if the EXECMOVE task has all the required arguments.
func (e *execOutputTask) Validate(params map[string]string) error {
	if _, _, _, err := parseExecArgs(params); err != nil {
		return fmt.Errorf("failed to parse task arguments: %w", err)
	}

	return nil
}

func getNewFileName(output string) string {
	lines := strings.Split(strings.TrimSpace(output), lineSeparator)
	lastLine := lines[len(lines)-1]

	if filename, ok := strings.CutPrefix(lastLine, "NEWFILENAME:"); ok {
		return filename
	}

	return ""
}

// Run executes the task by executing an external program with the given parameters.
func (e *execOutputTask) Run(parent context.Context, params map[string]string,
	_ *database.DB, logger *log.Logger, transCtx *model.TransferContext, _ any,
) error {
	output, runErr := runExec(parent, logger, transCtx, params)
	msg := ""
	if output != nil {
		msg = output.String()
	}

	if runErr != nil {
		if newPath := getNewFileName(msg); newPath != "" {
			transCtx.Transfer.LocalPath = newPath
			transCtx.Transfer.RemotePath = path.Join(path.Dir(
				transCtx.Transfer.RemotePath), path.Base(newPath))
		}

		return runErr
	}

	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		logger.Debug(scanner.Text())
	}

	logger.Debugf("Done executing command %s %s", params["path"], params["args"])

	return nil
}
