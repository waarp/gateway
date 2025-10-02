package backends

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/backends/s3"
)

//nolint:gochecknoinits //init is used by design
func init() {
	fs.Register("s3", s3.NewFS)
}
