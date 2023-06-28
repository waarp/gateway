//go:build unix

package fs

import (
	"github.com/hack-pad/hackpadfs/os"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

func NewLocalFS(*types.URL) (FS, error) {
	return &LocalFS{FS: os.NewFS()}, nil
}
