package utils

import (
	"os"
	"strings"
)

func ContainsFlag(flags, candidate int) bool {
	return flags&candidate == candidate
}

func DescribeFlags(flags int) string {
	var description []string

	switch flags & (os.O_RDONLY | os.O_WRONLY | os.O_RDWR) {
	case os.O_RDONLY:
		description = append(description, "read-only")
	case os.O_WRONLY:
		description = append(description, "write-only")
	case os.O_RDWR:
		description = append(description, "read-write")
	}

	if flags&os.O_APPEND != 0 {
		description = append(description, "append")
	}

	if flags&os.O_CREATE != 0 {
		description = append(description, "create")
	}

	if flags&os.O_EXCL != 0 {
		description = append(description, "exclusive")
	}

	if flags&os.O_SYNC != 0 {
		description = append(description, "sync")
	}

	if flags&os.O_TRUNC != 0 {
		description = append(description, "truncate")
	}

	return strings.Join(description, ", ")
}
