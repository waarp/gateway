//go:build !windows
// +build !windows

package tasks

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

const lineSeparator = "\n"

func getCommand(path, args string) *exec.Cmd {
	//nolint:gosec // Arguments cannot be passed outside a variable
	return exec.Command("/bin/sh", "-c", "exec "+path+" "+args)
}

func haltExec(cmd *exec.Cmd, ctx context.Context) error {
	haltRes := make(chan error)

	go func() {
		defer close(haltRes)

		haltRes <- cmd.Process.Signal(os.Interrupt)
	}()
	select {
	case err := <-haltRes:
		if err != nil {
			return fmt.Errorf("failed to halt process: %w", err)
		}

		return nil

	case <-ctx.Done():
		if killErr := cmd.Process.Kill(); killErr != nil {
			return fmt.Errorf("failed to halt process: %w", killErr)
		}

		return nil
	}
}
