package wg

import (
	"fmt"
	"strings"
)

type printLine struct {
	title string
	val   any
}

type printBuilder struct {
	indent  string
	builder strings.Builder
}

func newPrintBuilder(indent, header string) *printBuilder {
	p := &printBuilder{indent: indent}

	p.builder.WriteString(header)
	p.builder.WriteRune('\n')

	return p
}

func (p *printBuilder) addJustTitl(title string) {
	p.builder.WriteString(p.indent)
	p.builder.WriteString(title)
	p.builder.WriteRune('\n')
}

func (p *printBuilder) addTitlCond(title string, cond bool) {
	if cond {
		p.addJustTitl(title)
	}
}

func (p *printBuilder) addLineFull(title string, val any) {
	p.builder.WriteString(p.indent)
	p.builder.WriteString(title)
	p.builder.WriteRune(' ')
	p.builder.WriteString(fmt.Sprint(val))
	p.builder.WriteRune('\n')
}

func (p *printBuilder) addLineCond(title string, val any, cond bool) {
	if cond {
		p.addLineFull(title, val)
	}
}

func (p *printBuilder) addWithDefV(title string, val any, cond bool, defaultVal any) {
	if cond {
		p.addLineFull(title, val)
	} else {
		p.addLineFull(title, defaultVal)
	}
}

func (p *printBuilder) string() string { return p.builder.String() }
