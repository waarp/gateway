package types

import (
	"path/filepath"
	"strings"
)

func (u *URL) toOSPath() string {
	return filepath.FromSlash(strings.TrimPrefix(u.String(), "file:/"))
}
