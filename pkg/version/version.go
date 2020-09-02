// Package version contains miscellaneous elements about the version and the
// build of the software.
package version

//nolint:gochecknoglobals //cannot be constant: must be changed by the linter
var (
	Num    = "dev"
	Date   = ""
	Commit = "HEAD"
)
