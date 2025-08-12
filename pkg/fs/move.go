package fs

import (
	fsop "github.com/rclone/rclone/fs/operations"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
)

func MoveFile(srcPath, dstPath string) error {
	srcParsed, dstParsed, srcFs, dstFs, parsErr := parseSrcDstFs(srcPath, dstPath)
	if parsErr != nil {
		return parsErr
	}

	if srcParsed == dstParsed {
		return nil // nothing to do
	}

	if fsop.Same(srcFs.fs(), dstFs.fs()) {
		if err := fastRename(srcFs, srcParsed, dstParsed); err == nil {
			return nil
		}
	}

	return fallbackMove(srcFs, dstFs, srcParsed, dstParsed)
}

func fastRename(srcFs FS, srcParsed, dstParsed *parsedPath) error {
	mkdirPerms := conf.GlobalConfig.Paths.DirPerms

	if err := srcFs.MkdirAll(dstParsed.dir().Path, mkdirPerms); err != nil {
		return pathError("mkdir", dstParsed.dir().String(), err)
	}

	if err := srcFs.Rename(srcParsed.Path, dstParsed.Path); err != nil {
		return linkError("rename", srcParsed.String(), dstParsed.String(), err)
	}

	return nil
}

func fallbackMove(srcFs, dstFs FS, srcParsed, dstParsed *parsedPath) error {
	if err := doCopy(srcFs, dstFs, srcParsed, dstParsed); err != nil {
		return err
	}

	if err := srcFs.Remove(srcParsed.Path); err != nil {
		return pathError("remove", srcParsed.String(), err)
	}

	return nil
}
