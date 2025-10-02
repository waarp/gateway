package webfs

import (
	"io/fs"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/frontend"
)

//nolint:gochecknoglobals //must be var, changed for tests
var WebFS fs.FS = frontend.EmbedFS
