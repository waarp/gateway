package fs

import (
	"errors"
	"fmt"
	"io"
	gofs "io/fs"
	"path"
	"path/filepath"
	"strings"

	"github.com/hack-pad/hackpadfs"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/fs/flags"
)

func TrimPath(url *types.URL) string {
	fsPath := strings.Trim(url.Path, "/")
	if vol := filepath.VolumeName(fsPath); vol != "" {
		fsPath = strings.TrimPrefix(fsPath, vol+"/")
	}

	return path.Clean(fsPath)
}

// OpenFile opens and returns the given file with the given flags and permissions.
func OpenFile(name *types.URL, flag int, perm FileMode) (File, error) {
	fileSystem, fsErr := getFileSystem(name)
	if fsErr != nil {
		return nil, fsErr
	}

	file, err := hackpadfs.OpenFile(fileSystem, TrimPath(name), flag, perm)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

// Open opens and returns the given file in read-only mode.
func Open(name *types.URL) (File, error) {
	return OpenFile(name, flags.ReadOnly, 0o600)
}

// Create creates and returns the given file. If the file already exists, it is
// truncated.
func Create(name *types.URL) (File, error) {
	return OpenFile(name, flags.ReadWrite|flags.Create|flags.Truncate, 0o600)
}

// ReadFile reads the whole content of the given file, and returns it.
func ReadFile(name *types.URL) ([]byte, error) {
	file, opErr := Open(name)
	if opErr != nil {
		return nil, opErr
	}
	defer file.Close() //nolint:errcheck //Close never returns an error after reading

	content, rErr := io.ReadAll(file)
	if rErr != nil {
		return nil, fmt.Errorf("failed to read file content: %w", rErr)
	}

	return content, nil
}

// WriteFullFile writes the given content to the given file. If the file does not
// exist, it is created. If the file does exist, it is truncated, and its content
// overwritten.
func WriteFullFile(name *types.URL, data []byte) error {
	fileSystem, fsErr := getFileSystem(name)
	if fsErr != nil {
		return fsErr
	}

	if err := hackpadfs.WriteFullFile(fileSystem, TrimPath(name), data, 0o600); err != nil {
		return fmt.Errorf("failed to write file content: %w", err)
	}

	return nil
}

// MkdirAll creates the given directory and all of its parents (if they don't
// already exist).
func MkdirAll(name *types.URL) error {
	fileSystem, fsErr := getFileSystem(name)
	if fsErr != nil {
		return fsErr
	}

	file := TrimPath(name)
	if file == "" || file == "." || file == "/" {
		return nil // nothing to create
	}

	if err := hackpadfs.MkdirAll(fileSystem, file, 0o700); err != nil {
		if errors.Is(err, ErrExist) {
			// Even though it should, the hackpadfs.MkdirAll function does not
			// correctly ignore the ErrExist error returned by Mkdir when the
			// directory already exists, so we ignore it ourselves.
			return nil
		}

		return fmt.Errorf("failed to create the directories: %w", err)
	}

	return nil
}

// Stat returns a GenericFileInfo describing the given file.
func Stat(name *types.URL) (FileInfo, error) {
	fileSystem, fsErr := getFileSystem(name)
	if fsErr != nil {
		return nil, fsErr
	}

	info, statErr := hackpadfs.Stat(fileSystem, TrimPath(name))
	if statErr != nil {
		return nil, fmt.Errorf("failed to retrieve the file's info: %w", statErr)
	}

	return info, nil
}

// Rename renames (moves) oldname to newname. If newname already exists and is
// not a directory, Rename replaces it.
func Rename(oldname, newname *types.URL) error {
	fileSystem, fsErr := getFileSystem(oldname)
	if fsErr != nil {
		return fsErr
	}

	if err := hackpadfs.Rename(fileSystem, TrimPath(oldname), TrimPath(newname)); err != nil {
		return fmt.Errorf("failed to rename the file: %w", err)
	}

	return nil
}

// Remove removes the named file or directory.
func Remove(name *types.URL) error {
	fileSystem, fsErr := getFileSystem(name)
	if fsErr != nil {
		return fsErr
	}

	if err := hackpadfs.Remove(fileSystem, TrimPath(name)); err != nil {
		return fmt.Errorf("failed to delete the file: %w", err)
	}

	return nil
}

func ReadDir(name *types.URL) ([]DirEntry, error) {
	fileSystem, fsErr := getFileSystem(name)
	if fsErr != nil {
		return nil, fsErr
	}

	dirs, err := hackpadfs.ReadDir(fileSystem, TrimPath(name))
	if err != nil {
		return nil, fmt.Errorf("failed to read the directory: %w", err)
	}

	return dirs, nil
}

func Glob(pattern *types.URL) ([]*types.URL, error) {
	fileSystem, fsErr := getFileSystem(pattern)
	if fsErr != nil {
		return nil, fsErr
	}

	matches, globErr := gofs.Glob(fileSystem, TrimPath(pattern))
	if globErr != nil {
		return nil, fmt.Errorf("failed to walk the dir: %w", globErr)
	}

	urls := make([]*types.URL, len(matches))
	root := *pattern
	root.Path = "/"

	for i, match := range matches {
		urls[i] = root.JoinPath(match)
	}

	return urls, nil
}
