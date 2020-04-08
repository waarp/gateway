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

func whiteBold(text ...string) string {
	return fmt.Sprintf("\033[37;1m%s\033[0m", strings.Join(text, ""))
}

func whiteBoldUL(text ...string) string {
	return fmt.Sprintf("\033[37;1;4m%s\033[0m", strings.Join(text, ""))
}

func white(text ...string) string {
	return fmt.Sprintf("\033[37m%s\033[0m", strings.Join(text, ""))
}

func yellow(text ...string) string {
	return fmt.Sprintf("\033[33m%s\033[0m", strings.Join(text, ""))
}

func yellowBold(text ...string) string {
	return fmt.Sprintf("\033[33;1m%s\033[0m", strings.Join(text, ""))
}

func yellowBoldUL(text ...string) string {
	return fmt.Sprintf("\033[33;1;4m%s\033[0m", strings.Join(text, ""))
}

func redBold(text ...string) string {
	return fmt.Sprintf("\033[31;1m%s\033[0m", strings.Join(text, ""))
}

func greenBold(text ...string) string {
	return fmt.Sprintf("\033[32;1m%s\033[0m", strings.Join(text, ""))
}
