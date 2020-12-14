package main

import (
	"fmt"
	"os"
	"time"
)

type logger struct {
	path string
	pid  int
}

func newLogger(path string) (*logger, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("cannot open logfile: %w", err)
	}
	defer f.Close()

	l := &logger{
		path: path,
		pid:  os.Getpid(),
	}

	return l, nil
}

func (l logger) Print(msg string) {
	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}

	defer f.Close()

	fmt.Fprintf(f, "%s [%d] %s\n", time.Now().Format(time.RFC3339), l.pid, msg)
	fmt.Printf("%s [%d] %s\n", time.Now().Format(time.RFC3339), l.pid, msg)
}

func (l logger) Printf(msg string, args ...interface{}) {
	l.Print(fmt.Sprintf(msg, args...))
}
