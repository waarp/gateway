package filewatcher

import (
	"fmt"
	"io/fs"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ftp"
)

type ftpLister struct {
	logger    *log.Logger
	overrides *conf.ConfigOverride
	ctx       *model.TransferContext
}

func newFTPLister(logger *log.Logger, ctx *model.TransferContext, overrides *conf.ConfigOverride,
) *ftpLister {
	return &ftpLister{logger: logger, overrides: overrides, ctx: ctx}
}

func (f *ftpLister) List(pattern string) ([]fs.FileInfo, error) {
	client, connErr := ftp.Connect(f.logger, f.ctx, f.overrides)
	if connErr != nil {
		return nil, fmt.Errorf("failed to instantiate FTP client: %w", connErr)
	}

	defer func() {
		if err := client.Close(); err != nil {
			f.logger.Warningf("failed to close FTP connection: %v", err)
		}
	}()

	return list(client, f.ctx.Rule, pattern)
}
