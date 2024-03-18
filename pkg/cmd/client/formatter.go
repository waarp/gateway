package wg

import (
	"fmt"
	"io"
	"os"
	"reflect"

	listing "github.com/jedib0t/go-pretty/v6/list"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/mattn/go-colorable"
	"golang.org/x/term"
)

type Formatter struct {
	list listing.Writer
	out  io.Writer
}

func asColorable(w io.Writer) io.Writer {
	if file, ok := w.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		return colorable.NewColorable(file)
	}

	return colorable.NewNonColorable(w)
}

func NewFormatter(out io.Writer) *Formatter {
	l := listing.NewWriter()
	l.SetStyle(listing.StyleConnectedRounded)

	return &Formatter{
		list: l,
		out:  asColorable(out),
	}
}

func (f *Formatter) Indent()   { f.list.Indent() }
func (f *Formatter) UnIndent() { f.list.UnIndent() }
func (f *Formatter) Render()   { fmt.Fprintln(f.out, f.list.Render()) }
func (f *Formatter) Reset()    { f.list.Reset() }

func (f *Formatter) MainTitle(format string, args ...any) {
	fmt.Fprintf(f.out, format, args...)
	fmt.Fprintln(f.out)
}

func (f *Formatter) Println(format string, args ...any) {
	f.list.AppendItem(fmt.Sprintf(format, args...))
}

func (f *Formatter) Title(format string, args ...any) {
	colors := text.Colors{text.Bold, text.FgHiYellow}

	f.list.AppendItem(colors.Sprintf(format, args...))
}

func (f *Formatter) line(valColor text.Color, property string, value any) {
	valStr := ""
	if value != nil {
		valStr = valColor.Sprint(value)
	}

	f.list.AppendItem(text.FgHiYellow.Sprint(property) + ": " + valStr)
}

func (f *Formatter) Empty(property string, value any) {
	f.line(text.FgHiBlack, property, value)
}

func (f *Formatter) Error(property string, value any) {
	f.line(text.FgRed, property, value)
}

func (f *Formatter) Value(property string, value any) {
	f.line(text.FgWhite, property, value)
}

func (f *Formatter) ValueWithDefault(property string, value, defaultValue any) {
	if !reflect.ValueOf(value).IsZero() {
		f.Value(property, value)
	} else {
		f.Empty(property, defaultValue)
	}
}

func (f *Formatter) ValueCond(property string, value any) {
	if !reflect.ValueOf(value).IsZero() {
		f.line(text.FgWhite, property, value)
	}
}
