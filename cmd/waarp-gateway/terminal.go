package main

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

func isTerminal() bool {
	fIn, ok1 := in.(*os.File)
	fOut, ok2 := out.(*os.File)
	return ok1 && ok2 && term.IsTerminal(int(fIn.Fd())) && term.IsTerminal(int(fOut.Fd()))
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
	st, err := term.MakeRaw(int(in.(*os.File).Fd()))
	if err != nil {
		return "", err
	}
	defer func() { _ = term.Restore(int(in.(*os.File).Fd()), st) }()

	terminal := term.NewTerminal(in.(*os.File), "")
	pwd, err := terminal.ReadPassword("")
	if err != nil {
		return "", err
	}
	fmt.Fprintln(out)

	return pwd, nil
}
