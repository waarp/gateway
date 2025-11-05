//go:build !windows

package tasks

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

const (
	lineSeparator        = "\n"
	execInterruptTimeout = 5 * time.Second
)

func haltExec(cmd *exec.Cmd) error {
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
	case <-time.After(execInterruptTimeout):
		if killErr := cmd.Process.Kill(); killErr != nil {
			return fmt.Errorf("failed to halt process: %w", killErr)
		}

		return nil
	}
}
