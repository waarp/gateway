package fs

import "strings"

// ContainsFlags returns true if the flags mask contain the candidate flag.
func ContainsFlags(flags, candidate int) bool {
	return flags&candidate != 0
}

func DescribeFlags(flags int) string {
	var description []string

	switch flags & (FlagROnly | FlagWOnly | FlagRW) {
	case FlagROnly:
		description = append(description, "read-only")
	case FlagWOnly:
		description = append(description, "write-only")
	case FlagRW:
		description = append(description, "read-write")
	}

	if flags&FlagAppend != 0 {
		description = append(description, "append")
	}

	if flags&FlagCreate != 0 {
		description = append(description, "create")
	}

	if flags&FlagExclusive != 0 {
		description = append(description, "exclusive")
	}

	if flags&FlagSync != 0 {
		description = append(description, "sync")
	}

	if flags&FlagTruncate != 0 {
		description = append(description, "truncate")
	}

	return strings.Join(description, ", ")
}
