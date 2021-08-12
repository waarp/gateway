package main

import (
	"fmt"
	"runtime"

	"code.waarp.fr/apps/gateway/gateway/pkg/version"
)

type versionCommand struct {
}

func (d *versionCommand) Execute([]string) error {
	_, err := fmt.Printf("Waarp Gatewayd %s (%s - %s) [%s %s/%s %s]\n",
		version.Num, version.Date, version.Commit,
		runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.Compiler)

	return err
}
