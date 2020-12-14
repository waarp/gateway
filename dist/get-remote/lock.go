package main

import (
	"io/ioutil"
	"os"
)

type lock struct {
	path string
}

func (l lock) acquire() error {
	return ioutil.WriteFile(l.path, []byte("lock"), 0o600)
}

func (l lock) release() error {
	return os.RemoveAll(l.path)
}

func (l lock) isLocked() bool {
	return pathExists(l.path)
}
