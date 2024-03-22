//nolint:unparam //color functions are not always called with arguments
package wg

import (
	"io"
	"os"

	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/mattn/go-colorable"
	"golang.org/x/term"
)

func makeColorable(w io.Writer) io.Writer {
	if file, ok := w.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		return colorable.NewColorable(file)
	}

	return colorable.NewNonColorable(w)
}

// Deprecated: TODO: replace by makeColorable.
func getColorable() io.Writer {
	if file, ok := out.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		return colorable.NewColorable(file)
	}

	return colorable.NewNonColorable(out)
}

func bold(f string, a ...any) string   { return text.Bold.Sprintf(f, a...) }
func orange(f string, a ...any) string { return text.FgYellow.Sprintf(f, a...) }
func yellow(f string, a ...any) string { return text.FgHiYellow.Sprintf(f, a...) }
func red(f string, a ...any) string    { return text.FgRed.Sprintf(f, a...) }
func green(f string, a ...any) string  { return text.FgGreen.Sprintf(f, a...) }
func cyan(f string, a ...any) string   { return text.FgCyan.Sprintf(f, a...) }
func grey(f string, a ...any) string   { return text.FgHiBlack.Sprintf(f, a...) }

func boldOrange(f string, a ...any) string {
	return text.Colors{text.Bold, text.FgYellow}.Sprintf(f, a...)
}
