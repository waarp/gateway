package main

import (
	"fmt"
	"runtime"

	"code.waarp.fr/apps/gateway/gateway/pkg/version"
)

type versionCommand struct{}

//nolint:unparam,forbidigo // this is the intended behavior
func (d *versionCommand) Execute([]string) error {
	fmt.Printf("Waarp Gatewayd %s (%s - %s) [%s %s/%s %s]\n",
		version.Num, version.Date, version.Commit,
		runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.Compiler)

	return nil
}
