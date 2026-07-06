package filewatcher

import (
	"fmt"
	"strings"

	r66lib "code.waarp.fr/lib/r66"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
)

type r66Lister struct {
	logger    *log.Logger
	ctx       *model.TransferContext
	dialer    *protoutils.TraceDialer
	overrides *conf.ConfigOverride
}

func newR66Lister(logger *log.Logger, ctx *model.TransferContext,
	dialer *protoutils.TraceDialer, overrides *conf.ConfigOverride,
) *r66Lister {
	return &r66Lister{logger: logger, ctx: ctx, dialer: dialer, overrides: overrides}
}

func (r *r66Lister) List(pattern string) ([]fs.FileInfo, error) {
	conn, connErr := r66.OpenConn(r.logger, r.ctx, r.dialer, r.overrides)
	if connErr != nil {
		return nil, fmt.Errorf("failed to open R66 connection: %w", connErr)
	}

	defer func() {
		if err := conn.Close(); err != nil {
			r.logger.Warningf("failed to close R66 connection: %v", err)
		}
	}()

	ses, sesErr := conn.NewSession()
	if sesErr != nil {
		return nil, fmt.Errorf("failed to start R66 session: %w", sesErr)
	}

	var password []byte
	for _, cred := range r.ctx.RemoteAccountCreds {
		if cred.Type == auth.Password {
			password = []byte(cred.Value)
			break
		}
	}

	_, authErr := ses.Authent(r.ctx.RemoteAccount.Login, password, &r66lib.Config{})
	if authErr != nil {
		return nil, fmt.Errorf("failed to authenticate: %w", authErr)
	}

	defer ses.Close()

	resp, lsErr := ses.GetFileInfoV2(pattern, r.ctx.Rule.Name, r66lib.InfoListDetails)
	if lsErr != nil {
		return nil, fmt.Errorf("failed to list files: %w", lsErr)
	}

	files := make([]fs.FileInfo, 0, len(resp))

	for i := range resp {
		if !strings.EqualFold(resp[i].Type, "directory") {
			perm, err := fs.PermsFromString(resp[i].Permission)
			if err != nil {
				perm = 0o666
			}

			files = append(files, &fileInfo{
				name:    resp[i].Name,
				size:    resp[i].Size,
				mode:    perm,
				modTime: resp[i].LastModify,
			})
		}
	}

	return files, nil
}
