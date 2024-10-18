package utils

import (
	"path"
	"path/filepath"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
)

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
func GetPath(file string, elems ...Elem) (string, error) {
	file = filepath.ToSlash(file)
	if fs.IsAbsPath(file) {
		return file, nil
	}

	pathElems := []string{file}

	for _, elem := range elems {
		// skip empty elements
		if elem.str == "" || elem.str == "." {
			continue
		}

		// if we already have children, skip all leaves (they can't have children)
		if elem.isLeaf && len(pathElems) > 1 {
			continue
		}

		newElem := filepath.ToSlash(elem.str)
		pathElems = append([]string{newElem}, pathElems...)

		if fs.IsAbsPath(newElem) {
			if strings.HasSuffix(newElem, ":") {
				return newElem + path.Join(pathElems[1:]...), nil
			}

			return path.Join(pathElems...), nil
		}
	}

	return path.Join(pathElems...), nil
}
