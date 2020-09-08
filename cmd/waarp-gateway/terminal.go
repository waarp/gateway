package main

import (
	"fmt"

	"golang.org/x/crypto/ssh/terminal"
)

func isTerminal() bool {
	return terminal.IsTerminal(int(in.Fd())) && terminal.IsTerminal(int(out.Fd()))
}

func promptUser() (string, error) {
	if !isTerminal() {
		return "", fmt.Errorf("the username is missing from the URL")
	}

	var user string
	fmt.Fprintf(out, "Username: ")
	if _, err := fmt.Fscanln(in, &user); err != nil {
		return "", err
	}
	return user, nil
}

func promptPassword() (string, error) {
	if !isTerminal() {
		return "", fmt.Errorf("the user password is missing from the URL")
	}

	fmt.Fprint(out, "Password: ")
	st, err := terminal.MakeRaw(int(in.Fd()))
	if err != nil {
		return "", err
	}
	defer func() { _ = terminal.Restore(int(in.Fd()), st) }()

	term := terminal.NewTerminal(in, "")
	pwd, err := term.ReadPassword("")
	if err != nil {
		return "", err
	}
	fmt.Fprintln(out)

	return pwd, nil
}
