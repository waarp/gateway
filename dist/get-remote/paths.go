package main

import (
	"fmt"
	"os"
	"path/filepath"
)

type paths struct {
	baseDir, confDir string
}

func (p paths) lockFile() string {
	return filepath.Join(p.confDir, "get-remote.lock")
}

func (p paths) listFile() string {
	return filepath.Join(p.confDir, "get-files.list")
}

func getPaths() (paths, error) {
	execPath, err := filepath.Abs(os.Args[0])
	if err != nil {
		return paths{}, fmt.Errorf("cannot get executable path: %w", err)
	}

	execDir := filepath.Dir(execPath)
	parentDir := filepath.Dir(execDir)

	p := paths{}

	if pathExists(filepath.Join(parentDir, "etc")) {
		p.baseDir = parentDir
		p.confDir = filepath.Join(p.baseDir, "etc")
	} else if pathExists("/etc/waarp-gateway") {
		p.confDir = "/etc/waarp-gateway"
	}

	return p, nil
}

func pathExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}

	return true
}
