package wg

import (
	"fmt"
	"io"
	"os"

	"github.com/mattn/go-colorable"
	"golang.org/x/term"
)

//nolint:gochecknoglobals //global vars are needed here
var (
	stdOutput io.Writer = os.Stdout

	stdinFd  = int(os.Stdin.Fd())
	stdoutFd = int(os.Stdout.Fd())
)

const NoColors = "WAARP-NO-COLORS"

//nolint:gochecknoinits //needed for global variables
func init() {
	if os.Getenv(NoColors) != "" {
		stdOutput = colorable.NewNonColorable(stdOutput)
	}
}

func isTerminal() bool {
	return term.IsTerminal(stdinFd) && term.IsTerminal(stdoutFd)
}

func promptUser() (string, error) {
	if !isTerminal() {
		return "", fmt.Errorf("the username is missing from the URL: %w", errBadArgs)
	}

	fmt.Fprintf(stdOutput, "Username: ")

	var user string
	if _, err := fmt.Scanln(&user); err != nil {
		return "", fmt.Errorf("cannot read username: %w", err)
	}

	return user, nil
}

func promptPassword() (string, error) {
	if !isTerminal() {
		return "", fmt.Errorf("the user password is missing from the URL: %w", errBadArgs)
	}

	fmt.Fprint(stdOutput, "Password: ")

	st, err := term.MakeRaw(stdinFd)
	if err != nil {
		return "", fmt.Errorf("cannot change terminal mode: %w", err)
	}

	defer term.Restore(stdinFd, st) //nolint:errcheck //error is irrelevant

	terminal := term.NewTerminal(os.Stdin, "")

	pwd, err := terminal.ReadPassword("")
	if err != nil {
		return "", fmt.Errorf("cannot read password: %w", err)
	}

	fmt.Fprintln(stdOutput)

	return pwd, nil
}
