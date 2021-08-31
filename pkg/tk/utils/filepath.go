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
	if filepath.IsAbs(file) || path.IsAbs(file) {
		return ToOSPath(file)
	}

	strings := []string{file}
	for _, e := range elems {
		if e.String() == "" {
			continue
		}
		if e.IsLeaf() && len(strings) > 1 {
			continue
		}

		p := ToOSPath(e.String())
		strings = append([]string{p}, strings...)
		if filepath.IsAbs(e.String()) || path.IsAbs(e.String()) {
			return ToOSPath(strings...)
		}
	}
	return string(filepath.Separator) + ToOSPath(strings...)
}

// ToStandardPath transforms a path into a valid "file" URI according to the RFC 8089.
func ToStandardPath(paths ...string) string {
	return filepath.ToSlash(filepath.Join(paths...))
}

// ToOSPath transforms a "file" URI into a valid path for the OS.
func ToOSPath(paths ...string) string {
	return filepath.FromSlash(filepath.Join(paths...))
}
