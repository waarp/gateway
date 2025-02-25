package fs

import (
	"context"

	fsop "github.com/rclone/rclone/fs/operations"
)

func CopyFile(srcPath, dstPath string) (err error) {
	srcParsed, dstParsed, srcFs, dstFs, err := parseSrcDstFs(srcPath, dstPath)
	if err != nil {
		return err
	}

	if srcParsed == dstParsed {
		return nil
	}

	return doCopy(srcFs, dstFs, srcParsed, dstParsed)
}

//nolint:contextcheck //it's fine not to pass context
func doCopy(srcFs, dstFs FS, srcParsed, dstParsed *parsedPath) error {
	if err := fsop.CopyFile(context.Background(), dstFs.fs(), srcFs.fs(),
		dstParsed.unrooted(), srcParsed.unrooted()); err != nil {
		return linkError("copy", srcParsed.String(), dstParsed.String(), err)
	}

	return nil
}
