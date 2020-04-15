package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/mattn/go-colorable"
	"golang.org/x/crypto/ssh/terminal"
)

func getColorable() io.Writer {
	if terminal.IsTerminal(int(out.Fd())) {
		return colorable.NewColorable(out)
	}
	return colorable.NewNonColorable(out)
}

func bold(a ...interface{}) string {
	return fmt.Sprintf("\033[1m%s\033[0m", strings.TrimSuffix(fmt.Sprintln(a...), "\n"))
}

func orange(a ...interface{}) string {
	return fmt.Sprintf("\033[33m%s\033[0m", strings.TrimSuffix(fmt.Sprintln(a...), "\n"))
}

func yellow(a ...interface{}) string {
	return fmt.Sprintf("\033[93m%s\033[0m", strings.TrimSuffix(fmt.Sprintln(a...), "\n"))
}

func red(a ...interface{}) string {
	return fmt.Sprintf("\033[31m%s\033[0m", strings.TrimSuffix(fmt.Sprintln(a...), "\n"))
}

func green(a ...interface{}) string {
	return fmt.Sprintf("\033[32m%s\033[0m", strings.TrimSuffix(fmt.Sprintln(a...), "\n"))
}

func cyan(a ...interface{}) string {
	return fmt.Sprintf("\033[36m%s\033[0m", strings.TrimSuffix(fmt.Sprintln(a...), "\n"))
}
