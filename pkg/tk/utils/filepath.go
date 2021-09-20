package utils

import (
	"path"
)

// Elems is a list of elements forming a full filepath. Each element of the path
// is represented as a pair of values.
// The first value of  the pair must be a string containing the actual path
// element. The second member of the pair must be a boolean specifying if the
// must be the last element of the path.
type Elems [][2]interface{}

// GetPath return the path given by joining the given tail with all the given
// parents in the order they are given. The function will stop at the first
// absolute path, and return the path formed by all the previous parents.
func GetPath(tail string, elems Elems) string {
	if path.IsAbs(tail) {
		return tail
	}

	filepath := []string{tail}

	for i := range elems {
		p, ok := elems[i][0].(string)
		if !ok || p == "" {
			continue
		}

		if elems[i][1].(bool) && len(filepath) > 1 {
			continue
		}

		p = NormalizePath(p)
		filepath = append([]string{p}, filepath...)

		if path.IsAbs(p) {
			return path.Join(filepath...)
		}
	}

	return "/" + path.Join(filepath...)
}
