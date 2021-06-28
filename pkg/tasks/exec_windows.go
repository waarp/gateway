// +build windows

package tasks

import (
	"os"
	"os/exec"
)

const lineSeparator = "\r\n"

func getCommand(path, args string) *exec.Cmd {
	return exec.Command("cmd.exe", "/C", path+" "+args) //nolint:gosec
}

func haltExec(cmd *exec.Cmd) {
	_ = cmd.Process.Signal(os.Kill)
}
