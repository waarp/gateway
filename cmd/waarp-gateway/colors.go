package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattn/go-colorable"
	"golang.org/x/term"
)

func getColorable() io.Writer {
	if file, ok := out.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		return colorable.NewColorable(file)
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
