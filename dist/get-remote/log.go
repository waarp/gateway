package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type logger struct {
	path string
	pid  int
}

func newLogger(path string) (*logger, error) {
	f, err := os.OpenFile(filepath.Clean(path), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("cannot open logfile: %w", err)
	}

	_ = f.Close() //nolint:errcheck // nothing to handle the error

	l := &logger{
		path: path,
		pid:  os.Getpid(),
	}

	return l, nil
}

func (l logger) Print(msg string) {
	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}

	defer func() { _ = f.Close() }() //nolint:errcheck,gosec // nothing to handle the error

	fmt.Fprintf(f, "%s [%d] %s\n", time.Now().Format(time.RFC3339Nano), l.pid, msg)
	//nolint:forbidigo // A logger should be able to print
	fmt.Printf("%s [%d] %s\n", time.Now().Format(time.RFC3339Nano), l.pid, msg)
}

func (l logger) Printf(msg string, args ...interface{}) {
	l.Print(fmt.Sprintf(msg, args...))
}
