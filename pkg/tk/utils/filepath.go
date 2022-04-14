package utils

import (
	"path"
	"path/filepath"
)

// Elem is the interface which represent an element of a path tree.
type Elem interface {
	IsLeaf() bool
	String() string
}

// Leaf is a leaf in a path tree. Thus, it cannot have any children.
type Leaf string

// IsLeaf returns whether the element is a leaf (and thus, cannot have
// children) or not. Always true for type Leaf.
func (Leaf) IsLeaf() bool     { return true }
func (l Leaf) String() string { return string(l) }

// Branch is a branch of a path tree. It can have 1 or no child.
type Branch string

// IsLeaf returns whether the element is a leaf (and thus, cannot have
// children) or not. Always false for type Branch.
func (Branch) IsLeaf() bool     { return false }
func (b Branch) String() string { return string(b) }

// GetPath return the path given by joining the given tail with all the given
// parents in the order they are given. The function will stop at the first
// absolute path, and return the path formed by all the previous parents.
func GetPath(file string, elems ...Elem) string {
	if filepath.IsAbs(file) {
		return file
	}

	strings := []string{file}

	for i := range elems {
		if elems[i].String() == "" || elems[i].String() == "." {
			continue
		}

		if elems[i].IsLeaf() && len(strings) > 1 {
			continue
		}

		p := ToOSPath(elems[i].String())
		strings = append([]string{p}, strings...)

		if filepath.IsAbs(elems[i].String()) {
			return filepath.Join(strings...)
		}
	}

	return filepath.Join(strings...)
}

// ToStandardPath transforms a path into a valid "file" URI according to the RFC 8089.
func ToStandardPath(paths ...string) string {
	fullpath := filepath.Join(paths...)
	if filepath.IsAbs(fullpath) {
		return path.Join("/", filepath.ToSlash(fullpath))
	}

	return filepath.ToSlash(fullpath)
}

// ToOSPath transforms a "file" URI into a valid path for the OS.
func ToOSPath(paths ...string) string {
	return filepath.FromSlash(filepath.Join(paths...))
}
