package main

import (
	"fmt"
	"os"
	"path/filepath"
)

type paths struct {
	baseDir, logDir, confDir, binDir string
}

func (p paths) lockFile() string {
	return filepath.Join(p.logDir, "get-remote.lock")
}

func (p paths) logFile() string {
	return filepath.Join(p.logDir, "get-remote.log")
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
		p.logDir = filepath.Join(p.baseDir, "log")
		p.confDir = filepath.Join(p.baseDir, "etc")
		p.binDir = filepath.Join(p.baseDir, "bin")
	} else if pathExists("/etc/waarp-gateway") {
		p.logDir = "/var/log/waarp-gateway"
		p.confDir = "/etc/waarp-gateway"
		p.binDir = "/usr/bin"
		//} else {
		//	return paths{}, errors.New("Unable to detect installation mode")
	}

	return p, nil
}

func pathExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}

	return true
}
