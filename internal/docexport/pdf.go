package docexport

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
)

// fonts are TrueType files with full Latin coverage (Czech included); the
// PDF embeds the glyphs it uses, so text stays selectable and searchable.
type fonts struct{ serif, serifB, serifI, serifBI, sansB, sans, mono string }

// findFont asks fontconfig for a TrueType file, then tries known paths.
// fpdf reads .ttf only (no .otf or .ttc collections).
func findFont(pattern string, fallbacks ...string) string {
	if out, err := exec.Command("fc-match", "-f", "%{file}", pattern).Output(); err == nil {
		if p := strings.TrimSpace(string(out)); strings.HasSuffix(strings.ToLower(p), ".ttf") && exists(p) {
			return p
		}
	}
	for _, p := range fallbacks {
		if exists(p) {
			return p
		}
	}
	return ""
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

func findFonts() (fonts, error) {
	n, l, d := "/usr/share/fonts/noto/", "/usr/share/fonts/liberation/", "/usr/share/fonts/TTF/"
	f := fonts{
		serif:   findFont("Noto Serif:style=Regular", n+"NotoSerif-Regular.ttf", l+"LiberationSerif-Regular.ttf", d+"DejaVuSerif.ttf"),
		serifB:  findFont("Noto Serif:style=Bold", n+"NotoSerif-Bold.ttf", l+"LiberationSerif-Bold.ttf", d+"DejaVuSerif-Bold.ttf"),
		serifI:  findFont("Noto Serif:style=Italic", n+"NotoSerif-Italic.ttf", l+"LiberationSerif-Italic.ttf", d+"DejaVuSerif-Italic.ttf"),
		serifBI: findFont("Noto Serif:style=Bold Italic", n+"NotoSerif-BoldItalic.ttf", l+"LiberationSerif-BoldItalic.ttf", d+"DejaVuSerif-BoldItalic.ttf"),
		sans:    findFont("Noto Sans:style=Regular", n+"NotoSans-Regular.ttf", l+"LiberationSans-Regular.ttf", d+"DejaVuSans.ttf"),
		sansB:   findFont("Noto Sans:style=Bold", n+"NotoSans-Bold.ttf", l+"LiberationSans-Bold.ttf", d+"DejaVuSans-Bold.ttf"),
		mono:    findFont("Noto Sans Mono:style=Regular", n+"NotoSansMono-Regular.ttf", l+"LiberationMono-Regular.ttf", d+"DejaVuSansMono.ttf"),
	}
	for _, p := range []string{f.serif, f.serifB, f.serifI, f.serifBI, f.sans, f.sansB, f.mono} {
		if p == "" {
			return f, errors.New("no TrueType fonts found for PDF export (install noto-fonts or ttf-liberation)")
		}
	}
	return f, nil
}

const (
	pageMargin = 20.0
	bodySize   = 11.0
	lineH      = 5.6
)

// PDF writes the markdown document as an A4 PDF with selectable text.
func PDF(w io.Writer, title, markdown string) (err error) {
	// fpdf panics on some input (it did on emoji past U+FFFF): report it
	// as an error instead of taking the caller down.
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("PDF export failed: %v", p)
		}
	}()
	title, markdown = pdfSafe(title), pdfSafe(markdown)
	f, err := findFonts()
	if err != nil {
		return err
	}
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(pageMargin, pageMargin, pageMargin)
	pdf.SetAutoPageBreak(true, pageMargin)
	// fpdf resolves font paths against its own font folder, so the files are
	// read here and passed as bytes.
	for _, ff := range []struct{ family, style, path string }{
		{"serif", "", f.serif}, {"serif", "B", f.serifB}, {"serif", "I", f.serifI}, {"serif", "BI", f.serifBI},
		{"sans", "", f.sans}, {"sans", "B", f.sansB}, {"mono", "", f.mono},
	} {
		b, err := os.ReadFile(ff.path)
		if err != nil {
			return err
		}
		pdf.AddUTF8FontFromBytes(ff.family, ff.style, b)
	}
	if err := pdf.Error(); err != nil {
		return fmt.Errorf("loading fonts: %w", err)
	}
	pdf.SetTitle(title, true)
	pdf.SetCreator("NØX-Local", true)
	pdf.SetFooterFunc(func() {
		pdf.SetY(-12)
		pdf.SetFont("sans", "", 8)
		pdf.SetTextColor(140, 140, 140)
		pdf.CellFormat(0, 5, fmt.Sprintf("%d", pdf.PageNo()), "", 0, "C", false, 0, "")
	})
	pdf.AddPage()
	r := &pdfDoc{p: pdf}
	if title != "" {
		pdf.SetFont("sans", "B", 20)
		pdf.SetTextColor(31, 41, 55)
		pdf.MultiCell(0, 9, title, "", "L", false)
		pdf.SetFont("sans", "", 9)
		pdf.SetTextColor(120, 120, 120)
		pdf.CellFormat(0, 6, time.Now().Format("2. 1. 2006"), "", 1, "L", false, 0, "")
		pdf.Ln(4)
	}
	for _, b := range Parse(markdown) {
		r.block(b)
	}
	if err := pdf.Error(); err != nil {
		return err
	}
	return pdf.Output(w)
}

type pdfDoc struct{ p *fpdf.Fpdf }

func (r *pdfDoc) text() { r.p.SetTextColor(34, 34, 34) }

// pdfSafe drops what the embedded fonts cannot draw and fpdf cannot index:
// emoji and anything else past U+FFFF, variation selectors and joiners.
// A few common symbols get plain stand-ins first.
func pdfSafe(s string) string {
	s = symbolText.Replace(s)
	return strings.Map(func(r rune) rune {
		if r > 0xFFFF || r == 0xFE0F || r == 0xFE0E || r == 0x200D {
			return -1
		}
		return r
	}, s)
}

var symbolText = strings.NewReplacer("⭐", "*", "✅", "[x]", "❌", "[ ]", "⚠️", "!", "⚠", "!", "➡️", "->", "→", "->")

// noGlyph swaps characters the text fonts lack for plain equivalents.
var noGlyph = strings.NewReplacer("☑", "[x]", "☐", "[ ]")

// spans writes inline runs that wrap at the current margins.
func (r *pdfDoc) spans(ss []Span, size float64) {
	p := r.p
	for _, s := range ss {
		s.Text = noGlyph.Replace(s.Text)
		switch {
		case s.Code:
			p.SetFont("mono", "", size-1.5)
		default:
			style := ""
			if s.Bold {
				style += "B"
			}
			if s.Italic {
				style += "I"
			}
			p.SetFont("serif", style, size)
		}
		if s.Link != "" {
			p.SetTextColor(31, 95, 173)
			p.WriteLinkString(lineH, s.Text, s.Link)
			r.text()
			continue
		}
		p.Write(lineH, s.Text)
	}
}

func (r *pdfDoc) block(b Block) {
	p := r.p
	left, _, right, _ := p.GetMargins()
	r.text()
	switch b.Kind {
	case "heading":
		size := map[int]float64{1: 17, 2: 14, 3: 12.5}[b.Level]
		if size == 0 {
			size = 11.5
		}
		p.Ln(3)
		p.SetFont("sans", "B", size)
		p.SetTextColor(31, 41, 55)
		p.MultiCell(0, size*0.5, plain(b.Spans), "", "L", false)
		p.Ln(1.5)
	case "para":
		r.spans(b.Spans, bodySize)
		p.Ln(lineH + 2)
	case "quote":
		p.SetLeftMargin(left + 6)
		p.SetX(left + 6)
		y0 := p.GetY()
		p.SetTextColor(90, 90, 90)
		for i := range b.Spans {
			b.Spans[i].Italic = true
		}
		r.spans(b.Spans, bodySize)
		p.Ln(lineH)
		p.SetDrawColor(190, 190, 190)
		p.SetLineWidth(0.8)
		p.Line(left+2, y0+0.5, left+2, p.GetY()-0.5)
		p.SetLeftMargin(left)
		p.Ln(2)
	case "list":
		for _, it := range b.Items {
			indent := left + float64(it.Level)*6
			mark := "•"
			if it.Ordered {
				mark = fmt.Sprintf("%d.", it.Num)
			}
			p.SetX(indent)
			p.SetFont("serif", "", bodySize)
			p.CellFormat(6, lineH, mark, "", 0, "L", false, 0, "")
			p.SetLeftMargin(indent + 6)
			r.spans(it.Spans, bodySize)
			p.SetLeftMargin(left)
			p.Ln(lineH + 0.8)
		}
		p.Ln(1.5)
	case "code":
		p.SetFont("mono", "", 9)
		p.SetFillColor(243, 243, 243)
		p.SetTextColor(40, 40, 40)
		p.MultiCell(0, 4.6, strings.ReplaceAll(b.Code, "\t", "    "), "", "L", true)
		p.Ln(3)
	case "hr":
		y := p.GetY() + 2
		pw, _ := p.GetPageSize()
		p.SetDrawColor(200, 200, 200)
		p.SetLineWidth(0.3)
		p.Line(left, y, pw-right, y)
		p.Ln(6)
	case "table":
		r.table(b.Rows)
	}
}

func (r *pdfDoc) table(rows [][][]Span) {
	if len(rows) == 0 {
		return
	}
	p := r.p
	left, _, right, bottom := p.GetMargins()
	pw, ph := p.GetPageSize()
	cols := 0
	for _, row := range rows {
		cols = max(cols, len(row))
	}
	if cols == 0 {
		return
	}
	cw := (pw - left - right) / float64(cols)
	const pad, lh = 1.6, 4.6
	p.SetDrawColor(190, 190, 190)
	p.SetLineWidth(0.2)
	for i, row := range rows {
		style := ""
		if i == 0 {
			style = "B"
		}
		p.SetFont("serif", style, 9.5)
		cells := make([][]string, cols)
		height := 0.0
		for c := 0; c < cols; c++ {
			txt := ""
			if c < len(row) {
				txt = plain(row[c])
			}
			cells[c] = p.SplitText(txt, cw-2*pad)
			height = max(height, float64(max(len(cells[c]), 1))*lh+2*pad)
		}
		if p.GetY()+height > ph-bottom {
			p.AddPage()
		}
		y := p.GetY()
		for c := 0; c < cols; c++ {
			x := left + float64(c)*cw
			if i == 0 {
				p.SetFillColor(242, 242, 242)
				p.Rect(x, y, cw, height, "FD")
			} else {
				p.Rect(x, y, cw, height, "D")
			}
			for k, line := range cells[c] {
				p.SetXY(x+pad, y+pad+float64(k)*lh)
				p.CellFormat(cw-2*pad, lh, line, "", 0, "L", false, 0, "")
			}
		}
		p.SetXY(left, y+height)
	}
	p.Ln(4)
}
