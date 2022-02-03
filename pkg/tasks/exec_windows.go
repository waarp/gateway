//go:build windows
// +build windows

package tasks

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

const lineSeparator = "\r\n"

func getCommand(path, args string) *exec.Cmd {
	//nolint:gosec // Arguments cannot be passed outside a variable
	return exec.Command("cmd.exe", "/C", path+" "+args)
}

func haltExec(cmd *exec.Cmd, _ context.Context) error {
	if err := cmd.Process.Signal(os.Kill); err != nil {
		return fmt.Errorf("failed to halt process: %w", err)
	}
	return nil
}
