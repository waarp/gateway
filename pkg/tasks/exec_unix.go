// +build !windows

package tasks

import (
	"os"
	"os/exec"
	"time"
)

const lineSeparator = "\n"

func getCommand(path, args string) *exec.Cmd {
	return exec.Command("/bin/sh", "-c", "exec "+path+" "+args) //nolint:gosec
}

func haltExec(cmd *exec.Cmd) {
	timer := time.NewTimer(time.Second / 2)
	defer timer.Stop()
	intDone := make(chan struct{})
	go func() {
		defer close(intDone)
		_ = cmd.Process.Signal(os.Interrupt)
	}()
	select {
	case <-intDone:
	case <-timer.C:
		_ = cmd.Process.Kill()
	}
}
