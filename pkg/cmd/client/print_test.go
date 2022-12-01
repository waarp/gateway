package wg

import (
	"fmt"
	"strings"
)

type stringBuilder struct {
	indent  string
	builder strings.Builder
}

func newStringBuilder(indent, header string) *stringBuilder {
	p := &stringBuilder{indent: indent}

	p.builder.WriteString(header)
	p.builder.WriteRune('\n')

	return p
}

func (p *stringBuilder) addJustTitl(title string) {
	p.builder.WriteString(p.indent)
	p.builder.WriteString(title)
	p.builder.WriteRune('\n')
}

func (p *stringBuilder) addTitlCond(title string, cond bool) {
	if cond {
		p.addJustTitl(title)
	}
}

func (p *stringBuilder) addLineFull(title string, val any) {
	p.builder.WriteString(p.indent)
	p.builder.WriteString(title)
	p.builder.WriteRune(' ')
	p.builder.WriteString(fmt.Sprint(val))
	p.builder.WriteRune('\n')
}

func (p *stringBuilder) addLineCond(title string, val any, cond bool) {
	if cond {
		p.addLineFull(title, val)
	}
}

func (p *stringBuilder) addWithDefV(title string, val any, cond bool, defaultVal any) {
	if cond {
		p.addLineFull(title, val)
	} else {
		p.addLineFull(title, defaultVal)
	}
}

func (p *stringBuilder) string() string { return p.builder.String() }
