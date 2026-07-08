package filewatcher

import (
	"fmt"
	"io/fs"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
)

type sftpLister struct {
	logger    *log.Logger
	ctx       *model.TransferContext
	dialer    *protoutils.TraceDialer
	overrides *conf.ConfigOverride
}

func newSFTPLister(logger *log.Logger, ctx *model.TransferContext,
	dialer *protoutils.TraceDialer, overrides *conf.ConfigOverride,
) *sftpLister {
	return &sftpLister{logger: logger, ctx: ctx, dialer: dialer, overrides: overrides}
}

func (s *sftpLister) List(pattern string) ([]fs.FileInfo, error) {
	conn, connErr := sftp.OpenConn(s.logger, s.ctx, s.dialer, s.overrides)
	if connErr != nil {
		return nil, fmt.Errorf("failed to open SFTP connection: %w", connErr)
	}

	defer func() {
		if err := conn.Close(); err != nil {
			s.logger.Warningf("failed to close SFTP connection: %v", err)
		}
	}()

	return list(conn, s.ctx.Rule, pattern)
}
