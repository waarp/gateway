package filewatcher

import (
	"fmt"
	"io/fs"
	"path"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ftp"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/webdav"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
)

type UnsupportedProtocolError string

func (e UnsupportedProtocolError) Error() string {
	return fmt.Sprintf("unsupported protocol %q", string(e))
}

type Lister interface {
	List(pattern string) ([]fs.FileInfo, error)
}

func (s *Service) getLister(ctx *model.TransferContext) (Lister, error) {
	dialer, err := protoutils.NewDialerFor(ctx.Client.LocalAddress.String())
	if err != nil {
		return nil, fmt.Errorf("failed to create dialer: %w", err)
	}

	overrides := s.db.Config.Overrides

	switch ctx.Client.Protocol {
	case sftp.SFTP:
		return newSFTPLister(s.logger, ctx, dialer, overrides), nil
	case ftp.FTP, ftp.FTPS:
		return newFTPLister(s.logger, ctx, overrides), nil
	case r66.R66, r66.R66TLS:
		return newR66Lister(s.logger, ctx, dialer, overrides), nil
	case webdav.Webdav, webdav.WebdavTLS:
		return newWebdavLister(s.logger, ctx, overrides), nil
	case TestProtocol:
		if getTestLister != nil {
			return getTestLister(), nil
		}

		fallthrough
	default:
		return nil, UnsupportedProtocolError(ctx.Client.Protocol)
	}
}

type dirReader interface {
	ReadDir(string) ([]fs.FileInfo, error)
}

func list(cli dirReader, rule *model.Rule, pattern string) ([]fs.FileInfo, error) {
	dirPattern := path.Dir(pattern)
	filePattern := path.Base(pattern)

	fileInfos, listErr := cli.ReadDir(path.Join(rule.RemoteDir, dirPattern))
	if listErr != nil {
		return nil, fmt.Errorf("failed to list files: %w", listErr)
	}

	res := make([]fs.FileInfo, 0, len(fileInfos))

	for _, fi := range fileInfos {
		if !fi.IsDir() {
			ok, err := path.Match(filePattern, fi.Name())
			if err != nil {
				return nil, fmt.Errorf("bad pattern %q: %w", filePattern, err)
			}

			if ok {
				res = append(res, fi)
			}
		}
	}

	return res, nil
}

type fileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	sys     any
}

func (f *fileInfo) Name() string       { return f.name }
func (f *fileInfo) Size() int64        { return f.size }
func (f *fileInfo) Mode() fs.FileMode  { return f.mode }
func (f *fileInfo) ModTime() time.Time { return f.modTime }
func (f *fileInfo) IsDir() bool        { return f.mode.IsDir() }
func (f *fileInfo) Sys() any           { return f.sys }
