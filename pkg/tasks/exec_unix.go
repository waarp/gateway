//go:build aix || darwin || dragonfly || freebsd || (js && wasm) || linux || nacl || netbsd || openbsd || solaris

package tasks

import (
	"context"
	"os/exec"
)

func getCommand(ctx context.Context, path, args string) *exec.Cmd {
	//nolint:gosec // Arguments cannot be passed outside of a variable
	return exec.CommandContext(ctx, "/bin/sh", "-c", path+" "+args)
}
