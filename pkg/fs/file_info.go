package fs

import (
	"path"
	"time"
)

type GenericFileInfo struct {
	FileName    string
	FileSize    int64
	FileMode    FileMode
	LastModTime time.Time
	DataSource  any
}

func (f *GenericFileInfo) Name() string       { return path.Base(f.FileName) }
func (f *GenericFileInfo) Size() int64        { return f.FileSize }
func (f *GenericFileInfo) Mode() FileMode     { return f.FileMode }
func (f *GenericFileInfo) ModTime() time.Time { return f.LastModTime }
func (f *GenericFileInfo) IsDir() bool        { return f.FileMode.IsDir() }
func (f *GenericFileInfo) Sys() any           { return f.DataSource }
