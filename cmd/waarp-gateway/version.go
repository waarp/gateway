package main

import (
	"fmt"
	"runtime"
)

var (
	versionNum    = "dev"
	versionDate   = ""
	versionCommit = "HEAD"
)

type versionCommand struct {
}

func (d *versionCommand) Execute([]string) error {
	_, err := fmt.Printf("Waarp Gateway %s (%s - %s) [%s %s/%s %s]\n",
		versionNum, versionDate, versionCommit,
		runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.Compiler)

	return err
}
