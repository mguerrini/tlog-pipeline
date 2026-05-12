package common

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// XMLBuilder es un escritor XML manual minimalista. Usamos string-builder en
// vez de encoding/xml para tener control absoluto sobre el orden de campos,
// las tags vacías (<Foo/>) vs (<Foo></Foo>), y la indentación.
type XMLBuilder struct {
	sb     strings.Builder
	indent string
	stack  []string
}

// NewXMLBuilder construye un builder con la cabecera <?xml ... ?>.
func NewXMLBuilder() *XMLBuilder {
	x := &XMLBuilder{}
	x.sb.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	return x
}

// String devuelve el XML acumulado.
func (x *XMLBuilder) String() string { return x.sb.String() }

// Open abre una tag y aumenta el nivel de indentación.
func (x *XMLBuilder) Open(name string) {
	x.sb.WriteString(x.indent)
	x.sb.WriteString("<")
	x.sb.WriteString(name)
	x.sb.WriteString(">\n")
	x.stack = append(x.stack, name)
	x.indent += "  "
}

// Close cierra la última tag abierta.
func (x *XMLBuilder) Close() {
	if len(x.stack) == 0 {
		return
	}
	name := x.stack[len(x.stack)-1]
	x.stack = x.stack[:len(x.stack)-1]
	x.indent = x.indent[:len(x.indent)-2]
	x.sb.WriteString(x.indent)
	x.sb.WriteString("</")
	x.sb.WriteString(name)
	x.sb.WriteString(">\n")
}

// Element escribe <Name>value</Name> en una sola línea, escapando el valor.
// Si value es vacío, escribe <Name/>.
func (x *XMLBuilder) Element(name, value string) {
	x.sb.WriteString(x.indent)
	x.sb.WriteString("<")
	x.sb.WriteString(name)
	x.sb.WriteString(">")
	if value != "" {
		x.sb.WriteString(escape(value))
	}
	x.sb.WriteString("</")
	x.sb.WriteString(name)
	x.sb.WriteString(">\n")
}

// EmptyElement escribe <Name/>.
func (x *XMLBuilder) EmptyElement(name string) {
	x.sb.WriteString(x.indent)
	x.sb.WriteString("<")
	x.sb.WriteString(name)
	x.sb.WriteString(">")
	x.sb.WriteString("</")
	x.sb.WriteString(name)
	x.sb.WriteString(">\n")
}

// Comment escribe un comentario inline.
func (x *XMLBuilder) Comment(c string) {
	x.sb.WriteString(x.indent)
	x.sb.WriteString("<!-- ")
	x.sb.WriteString(c)
	x.sb.WriteString(" -->\n")
}

func escape(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	if err := xml.EscapeText(stringWriter{&b}, []byte(s)); err != nil {
		// fallback: devolver original si falla (no debería pasar con strings.Builder)
		return s
	}
	return b.String()
}

type stringWriter struct{ sb *strings.Builder }

func (w stringWriter) Write(p []byte) (int, error) { return w.sb.Write(p) }

// Sprintf-like helper.
func F(format string, args ...any) string { return fmt.Sprintf(format, args...) }
