package webdav

import (
	"context"
	"errors"
	"fmt"
	"io"
	gofs "io/fs"
	"mime"
	"net/http"
	"path"
	"path/filepath"

	"golang.org/x/net/webdav"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
)

const (
	propfindMethod = "PROPFIND"
)

func (w *webdavFS) propfind(name string) (webdav.File, error) {
	realPath, err := protoutils.GetRealPath(false, w.db, w.logger, w.server, w.account, name)
	if err != nil {
		return nil, fmt.Errorf("failed to build the file path: %w", err)
	} else if realPath == "" {
		dir, err2 := protoutils.GetRuleDir(w.db, w.server, w.account, name)
		if err2 != nil {
			return nil, fmt.Errorf("failed to get rule directory: %w", err2)
		}

		return dir, nil
	}

	file, err := fs.Open(realPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open %q: %w", realPath, err)
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

func (p *propFile) getContentType() (string, error) {
	name := path.Ext(filepath.ToSlash(p.f.Name()))

	ctype := mime.TypeByExtension(name)
	if ctype != "" {
		return ctype, nil
	}

	const bufSize = 512 // max size considered by http.DetectContentType
	buf := make([]byte, 0, bufSize)
	n, err := io.ReadFull(p.f, buf)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", fmt.Errorf("failed to read file %q: %w", p.f.Name(), err)
	}

	return http.DetectContentType(buf[:n]), nil
}

//nolint:wrapcheck //no need to wrap here
func (p *propFile) Close() error { return p.f.Close() }

//nolint:wrapcheck //no need to wrap here
func (p *propFile) Stat() (fs.FileInfo, error) {
	info, err := p.f.Stat()
	if err != nil {
		return nil, err
	}

	if info.IsDir() {
		return info, nil
	}

	ctype, err := p.getContentType()
	if err != nil {
		return nil, err
	}

	return &propInfo{FileInfo: info, ctype: ctype}, nil
}

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

type propInfo struct {
	fs.FileInfo

	ctype string
}

func (p *propInfo) ContentType(context.Context) (string, error) {
	return p.ctype, nil
}
