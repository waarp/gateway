package wg

import (
	"fmt"
	"runtime"

	"code.waarp.fr/apps/gateway/gateway/pkg/version"
)

type Version struct{}

//nolint:forbidigo,unparam // this is the intended behavior
func (*Version) Execute([]string) error {
	fmt.Printf("Waarp Gateway %s (%s - %s) [%s %s/%s %s]\n",
		version.Num, version.Date, version.Commit,
		runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.Compiler)

	return nil
}
