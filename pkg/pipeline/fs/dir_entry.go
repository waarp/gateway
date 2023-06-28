package fs

type GenericDirEntry struct {
	*GenericFileInfo
}

func (g *GenericDirEntry) Type() FileMode          { return g.FileMode.Type() }
func (g *GenericDirEntry) Info() (FileInfo, error) { return g.GenericFileInfo, nil }
