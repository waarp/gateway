package main

import "os"

type lock struct {
	path string
}

func (l lock) acquire() error {
	f, err := os.Create(l.path)
	if err != nil {
		return err
	}
	defer f.Close()

	return nil
}

func (l lock) release() error {
	return os.RemoveAll(l.path)
}

func (l lock) isLocked() bool {
	return pathExists(l.path)
}
