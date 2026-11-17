// Package docexport turns markdown into DOCX and PDF files: the documents
// the model writes (reports, notes) become files people can send on.
package docexport

import (
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// Span is a run of inline text with one style.
type Span struct {
	Text   string
	Bold   bool
	Italic bool
	Code   bool
	Link   string
}

// Item is one list item; Level counts nesting from 0.
type Item struct {
	Level   int
	Ordered bool
	Num     int
	Spans   []Span
}

// Block is one block of the document.
type Block struct {
	Kind  string // heading | para | list | code | quote | hr | table
	Level int    // heading level
	Spans []Span
	Items []Item
	Code  string
	Lang  string
	Rows  [][][]Span // table rows (first is the header), cells, spans
}

// Parse reads markdown (GitHub flavour: tables, task lists, strikethrough).
func Parse(src string) []Block {
	md := goldmark.New(goldmark.WithExtensions(extension.GFM))
	source := []byte(src)
	doc := md.Parser().Parse(text.NewReader(source))
	var out []Block
	for n := doc.FirstChild(); n != nil; n = n.NextSibling() {
		out = append(out, blocks(n, source)...)
	}
	return out
}

func blocks(n ast.Node, src []byte) []Block {
	switch v := n.(type) {
	case *ast.Heading:
		return []Block{{Kind: "heading", Level: v.Level, Spans: inlines(n, src, Span{})}}
	case *ast.Paragraph, *ast.TextBlock:
		return []Block{{Kind: "para", Spans: inlines(n, src, Span{})}}
	case *ast.FencedCodeBlock:
		return []Block{{Kind: "code", Code: lines(n, src), Lang: string(v.Language(src))}}
	case *ast.CodeBlock:
		return []Block{{Kind: "code", Code: lines(n, src)}}
	case *ast.Blockquote:
		var spans []Span
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			if len(spans) > 0 {
				spans = append(spans, Span{Text: "\n"})
			}
			spans = append(spans, inlines(c, src, Span{})...)
		}
		return []Block{{Kind: "quote", Spans: spans}}
	case *ast.ThematicBreak:
		return []Block{{Kind: "hr"}}
	case *ast.List:
		var items []Item
		listItems(v, src, 0, &items)
		return []Block{{Kind: "list", Items: items}}
	case *east.Table:
		var rows [][][]Span
		for r := n.FirstChild(); r != nil; r = r.NextSibling() {
			var cells [][]Span
			for c := r.FirstChild(); c != nil; c = c.NextSibling() {
				cells = append(cells, inlines(c, src, Span{}))
			}
			rows = append(rows, cells)
		}
		return []Block{{Kind: "table", Rows: rows}}
	case *ast.HTMLBlock:
		return nil
	}
	return []Block{{Kind: "para", Spans: inlines(n, src, Span{})}}
}

func listItems(l *ast.List, src []byte, level int, out *[]Item) {
	num := l.Start
	if num == 0 {
		num = 1
	}
	for li := l.FirstChild(); li != nil; li = li.NextSibling() {
		it := Item{Level: level, Ordered: l.IsOrdered(), Num: num}
		num++
		var nested []*ast.List
		for c := li.FirstChild(); c != nil; c = c.NextSibling() {
			if sub, ok := c.(*ast.List); ok {
				nested = append(nested, sub)
				continue
			}
			if len(it.Spans) > 0 {
				it.Spans = append(it.Spans, Span{Text: " "})
			}
			it.Spans = append(it.Spans, inlines(c, src, Span{})...)
		}
		*out = append(*out, it)
		for _, sub := range nested {
			listItems(sub, src, level+1, out)
		}
	}
}

func lines(n ast.Node, src []byte) string {
	var b strings.Builder
	ls := n.Lines()
	for i := 0; i < ls.Len(); i++ {
		seg := ls.At(i)
		b.Write(seg.Value(src))
	}
	return strings.TrimRight(b.String(), "\n")
}

// inlines flattens the inline children of n into styled spans.
func inlines(n ast.Node, src []byte, style Span) []Span {
	var out []Span
	add := func(s Span) {
		if s.Text == "" {
			return
		}
		if k := len(out) - 1; k >= 0 {
			p := out[k]
			if p.Bold == s.Bold && p.Italic == s.Italic && p.Code == s.Code && p.Link == s.Link {
				out[k].Text += s.Text
				return
			}
		}
		out = append(out, s)
	}
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch v := c.(type) {
		case *ast.Text:
			s := style
			s.Text = string(v.Segment.Value(src))
			add(s)
			if v.HardLineBreak() {
				add(Span{Text: "\n", Bold: style.Bold, Italic: style.Italic, Link: style.Link})
			} else if v.SoftLineBreak() {
				s.Text = " "
				add(s)
			}
		case *ast.String:
			s := style
			s.Text = string(v.Value)
			add(s)
		case *ast.CodeSpan:
			s := style
			s.Code = true
			var b strings.Builder
			for t := c.FirstChild(); t != nil; t = t.NextSibling() {
				if tt, ok := t.(*ast.Text); ok {
					b.Write(tt.Segment.Value(src))
				}
			}
			s.Text = b.String()
			add(s)
		case *ast.Emphasis:
			s := style
			if v.Level >= 2 {
				s.Bold = true
			} else {
				s.Italic = true
			}
			for _, sp := range inlines(c, src, s) {
				add(sp)
			}
		case *ast.Link:
			s := style
			s.Link = string(v.Destination)
			for _, sp := range inlines(c, src, s) {
				add(sp)
			}
		case *ast.AutoLink:
			s := style
			s.Link = string(v.URL(src))
			s.Text = string(v.Label(src))
			add(s)
		case *ast.Image:
			s := style
			s.Italic = true
			s.Text = "[image: " + plain(inlines(c, src, Span{})) + "]"
			add(s)
		case *east.TaskCheckBox:
			s := style
			s.Text = "☐ "
			if v.IsChecked {
				s.Text = "☑ "
			}
			add(s)
		case *ast.RawHTML:
			// dropped: no HTML in exported documents
		default:
			for _, sp := range inlines(c, src, style) {
				add(sp)
			}
		}
	}
	return out
}

// plain joins spans into unstyled text.
func plain(spans []Span) string {
	var b strings.Builder
	for _, s := range spans {
		b.WriteString(s.Text)
	}
	return b.String()
}
