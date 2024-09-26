package utils

import (
	"fmt"
	"path"
	"path/filepath"

	"github.com/rclone/rclone/fs/fspath"
)

func parsePath(filePath string) (string, string, bool, error) {
	if filePath == "" {
		return "", "", false, nil
	}

	if filepath.IsAbs(filePath) {
		return "", filepath.ToSlash(filePath), true, nil
	}

	if parsed, err := fspath.Parse(filePath); err != nil {
		return "", "", false, fmt.Errorf("failed to parse file path: %w", err)
	} else {
		return parsed.Name, parsed.Path, parsed.Name != "", nil
	}
}

type Elem struct {
	isLeaf bool
	str    string
}

// Leaf is a leaf in a path tree. Thus, it cannot have any children.
func Leaf(str string) Elem { return Elem{true, str} }

// Branch is a branch of a path tree. It can have 1 or no child.
func Branch(str string) Elem { return Elem{false, str} }

// GetPath return the path given by joining the given tail with all the given
// parents in the order they are given. The function will stop at the first
// absolute path, and return the path formed by all the previous parents.
func GetPath(file string, elems ...Elem) (string, string, error) {
	backend, fPath, isRooted, fErr := parsePath(file)
	if fErr != nil {
		return "", "", fErr
	} else if isRooted {
		return backend, fPath, nil
	}

	strings := []string{file}

	for _, elem := range elems {
		// skip empty elements
		if elem.str == "" || elem.str == "." {
			continue
		}

		// if we already have children, skip all leaves (they can't have children)
		if elem.isLeaf && len(strings) > 1 {
			continue
		}

		backend, fPath, isRooted, fErr = parsePath(elem.str)
		if fErr != nil {
			return "", "", fErr
		}

		strings = append([]string{fPath}, strings...)

		if isRooted {
			return backend, path.Join(strings...), nil
		}
	}

	return "", path.Join(strings...), nil
}
