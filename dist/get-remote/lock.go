package main

import (
	"fmt"
	"io/ioutil"
	"os"
)

type lock struct {
	path string
}

func (l lock) acquire() error {
	if err := ioutil.WriteFile(l.path, []byte("lock"), 0o600); err != nil {
		return fmt.Errorf("cannot create lock file: %w", err)
	}

	return nil
}

func (l lock) release() error {
	if err := os.RemoveAll(l.path); err != nil {
		return fmt.Errorf("cannot remove lock file: %w", err)
	}

	return nil
}

func (l lock) isLocked() bool {
	return pathExists(l.path)
}
