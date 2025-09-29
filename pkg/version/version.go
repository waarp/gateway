// Package version contains miscellaneous elements about the version and the
// build of the software.
package version

import "time"

//nolint:gochecknoglobals //cannot be constant: must be changed by the linker
var (
	Num    = "dev"
	Date   = time.Now().Format(time.RFC3339)
	Commit = "HEAD"
)
