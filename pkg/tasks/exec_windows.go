//go:build windows

package tasks

import (
	"fmt"
	"os/exec"
)

const lineSeparator = "\r\n"

func haltExec(cmd *exec.Cmd) error {
	if err := cmd.Process.Kill(); err != nil {
		return fmt.Errorf("failed to halt process: %w", err)
	}

	return nil
}
