package main

import (
	"fmt"

	"golang.org/x/term"
)

func isTerminal() bool {
	return term.IsTerminal(int(in.Fd())) && term.IsTerminal(int(out.Fd()))
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
	st, err := term.MakeRaw(int(in.Fd()))
	if err != nil {
		return "", err
	}
	defer func() { _ = term.Restore(int(in.Fd()), st) }()

	term := term.NewTerminal(in, "")
	pwd, err := term.ReadPassword("")
	if err != nil {
		return "", err
	}
	fmt.Fprintln(out)

	return pwd, nil
}
