package webdav

import (
	"fmt"
	gofs "io/fs"

	"golang.org/x/net/webdav"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
)

const (
	propfindMethod = "PROPFIND"
)

func (w *webdavFS) propfind(filepath string) (webdav.File, error) {
	realPath, err := protoutils.GetRealPath(false, w.db, w.logger, w.server, w.account, filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to build the file path: %w", err)
	} else if realPath == "" {
		dir, err2 := protoutils.GetRuleDir(w.db, w.server, w.account, filepath)
		if err2 != nil {
			return nil, fmt.Errorf("failed to get rule directory: %w", err2)
		}

		return dir, nil
	}

	file, err := fs.Open(realPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open directory %q: %w", realPath, err)
	}

	return prop(file), nil
}

var (
	ErrReadOnProp  = fmt.Errorf("read on %s request", propfindMethod)
	ErrWriteOnProp = fmt.Errorf("write on %s request", propfindMethod)
	ErrSeekOnProp  = fmt.Errorf("seek on %s request", propfindMethod)
)

type propFile struct {
	f fs.File
}

func prop(f fs.File) webdav.File { return &propFile{f: f} }

//nolint:wrapcheck //no need to wrap here
func (p *propFile) Close() error { return p.f.Close() }

//nolint:wrapcheck //no need to wrap here
func (p *propFile) Stat() (fs.FileInfo, error) { return p.f.Stat() }

func (p *propFile) Read([]byte) (int, error)       { return 0, ErrReadOnProp }
func (p *propFile) Write([]byte) (int, error)      { return 0, ErrWriteOnProp }
func (p *propFile) Seek(int64, int) (int64, error) { return 0, ErrSeekOnProp }

//nolint:wrapcheck //no need to wrap here
func (p *propFile) Readdir(n int) ([]fs.FileInfo, error) {
	entries, err := p.f.ReadDir(n)
	if err != nil {
		return nil, err
	}

	infos := make([]gofs.FileInfo, len(entries))
	for i, entry := range entries {
		infos[i], err = entry.Info()
		if err != nil {
			return nil, err
		}
	}

	return infos, nil
}
