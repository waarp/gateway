package filewatcher

import (
	"fmt"
	"io/fs"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/webdav"
)

type webdavLister struct {
	logger    *log.Logger
	ctx       *model.TransferContext
	overrides *conf.ConfigOverride
}

func newWebdavLister(logger *log.Logger, ctx *model.TransferContext,
	overrides *conf.ConfigOverride,
) *webdavLister {
	return &webdavLister{logger: logger, ctx: ctx, overrides: overrides}
}

func (w *webdavLister) List(pattern string) ([]fs.FileInfo, error) {
	client, err := webdav.GetClient(w.logger, w.ctx, w.overrides)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate WebDAV client: %w", err)
	}

	return list(client, w.ctx.Rule, pattern)
}
