package wgd

import (
	"fmt"
	"runtime"

	"code.waarp.fr/apps/gateway/gateway/pkg/version"
)

type VersionCommand struct{}

//nolint:unparam,forbidigo // this is the intended behavior
func (d *VersionCommand) Execute([]string) error {
	fmt.Printf("Waarp Gatewayd %s (%s - %s) [%s %s/%s %s]\n",
		version.Num, version.Date, version.Commit,
		runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.Compiler)

	return nil
}
