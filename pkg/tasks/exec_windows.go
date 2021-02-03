package tasks

import (
	"context"
	"os/exec"
)

const lineSeparator = "\r\n"

func getCommand(ctx context.Context, path, args string) *exec.Cmd {
	return exec.CommandContext(ctx, "cmd.exe", "/C", path+" "+args) //nolint:gosec
}
