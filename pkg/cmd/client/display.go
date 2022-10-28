package wg

import (
	"fmt"
	"io"
)

const NotApplicable = "N/A"

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
