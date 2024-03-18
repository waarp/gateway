package wg

import (
	"fmt"
	"io"
	"sort"
)

const NotApplicable = "N/A"

type pair struct {
	key string
	val any
}

func displayMap[T any](f *Formatter, title, emptyValue string, m map[string]T) {
	if len(m) == 0 {
		f.Empty(title, emptyValue)

		return
	}

	f.Title(title)
	f.Indent()

	defer f.UnIndent()

	pairs := make([]pair, 0, len(m))

	for key, val := range m {
		pairs = append(pairs, pair{key: key, val: val})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].key < pairs[j].key
	})

	for i := range pairs {
		f.Value(pairs[i].key, pairs[i].val)
	}
}

func writeLine(w io.Writer, key string, val any) {
	fmt.Fprintln(w, key, val)
}

func writeDefV(w io.Writer, key string, val any, cond bool, defaultVal any) {
	if cond {
		writeLine(w, key, val)
	} else {
		writeLine(w, key, defaultVal)
	}
}

func writeCond(w io.Writer, key string, val any, cond bool) {
	if cond {
		writeLine(w, key, val)
	}
}
