// +build aix darwin dragonfly freebsd js,wasm linux nacl netbsd openbsd solaris

package tasks

import (
	"context"
	"os/exec"
)

func getCommand(ctx context.Context, path, args string) *exec.Cmd {
	return exec.CommandContext(ctx, "/bin/sh", "-c", path+" "+args) //nolint:gosec
}
