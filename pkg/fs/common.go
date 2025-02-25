package fs

import "path/filepath"

func parseFs(path string) (parsed *parsedPath, vfs FS, err error) {
	parsed, err = parsePath(path)
	if err != nil {
		return nil, nil, pathError("parse", path, err)
	}

	if vfs, err = getFS(parsed); err != nil {
		return nil, nil, pathError("parse", path, err)
	}

	return
}

func parseSrcDstFs(srcPath, dstPath string) (srcParsed, dstParsed *parsedPath, srcFs, dstFs FS, err error) {
	srcParsed, srcFs, err = parseFs(srcPath)
	if err != nil {
		return
	}

	dstParsed, dstFs, err = parseFs(dstPath)
	if err != nil {
		return
	}

	return
}

func JoinPath(elems ...string) string {
	return filepath.ToSlash(filepath.Join(elems...))
}
