package fs

import "os"

func LockFile(file File) error {
	localFile, isLocal := file.(*os.File)
	if !isLocal {
		return nil
	}

	return lockFile(localFile)
}

func LockFileR(file File) error {
	localFile, isLocal := file.(*os.File)
	if !isLocal {
		return nil
	}

	return lockFileR(localFile)
}
