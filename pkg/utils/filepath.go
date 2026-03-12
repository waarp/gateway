package utils

import (
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
	if fs.IsAbsPath(file) {
		return fs.JoinPath(file), nil
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

		newElem := elem.str
		pathElems = append([]string{newElem}, pathElems...)

		if fs.IsAbsPath(newElem) {
			if strings.HasSuffix(newElem, ":") {
				return newElem + fs.JoinPath(pathElems[1:]...), nil
			}

			return fs.JoinPath(pathElems...), nil
		}
	}

	return fs.JoinPath(pathElems...), nil
}
