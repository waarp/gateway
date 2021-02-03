package utils

import (
	"path"
)

type Elem interface {
	IsLeaf() bool
	String() string
}

type Leaf string

func (Leaf) IsLeaf() bool     { return true }
func (l Leaf) String() string { return string(l) }

type Branch string

func (Branch) IsLeaf() bool     { return false }
func (b Branch) String() string { return string(b) }

// MakePath return the path given by joining the given tail with all the given
// parents in the order they are given. The function will stop at the first
// absolute path, and return the path formed by all the previous parents.
func GetPath(file string, elems ...Elem) string {
	if path.IsAbs(file) {
		return file
	}

	filepath := []string{file}
	for _, e := range elems {
		if e.String() == "" {
			continue
		}
		if e.IsLeaf() && len(filepath) > 1 {
			continue
		}

		p := NormalizePath(e.String())
		filepath = append([]string{p}, filepath...)
		if path.IsAbs(p) {
			return path.Join(filepath...)
		}
	}
	return "/" + path.Join(filepath...)
}
