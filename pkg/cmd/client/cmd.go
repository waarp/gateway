package wg

import (
	"io"
	"os"

	"github.com/mattn/go-colorable"
	"golang.org/x/term"
)

type commander interface{ execute(w io.Writer) error }

//nolint:wrapcheck //no need to wrap errors here
func execute(cmd commander) error {
	switch w := stdOutput.(type) {
	case *colorable.NonColorable:
		return cmd.execute(w)
	case *os.File:
		if term.IsTerminal(int(w.Fd())) {
			return cmd.execute(colorable.NewColorable(w))
		}
	default:
	}

	return cmd.execute(colorable.NewNonColorable(stdOutput))
}
