package docexport

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
)

const sample = `# Tepelná čerpadla 2026

Přehled **dotací** a *podmínek*. Zdroj: [MŽP](https://www.mzp.cz) a ` + "`kód`" + `.

## Kdo má nárok

- vlastníci rodinných domů
- obce
  - menší obce do 2 000 obyvatel
1. podat žádost
2. doložit fakturu

| Program | Výše |
|---|---|
| Kotlíková dotace | až [ČÁSTKA] Kč |
| NZÚ | podle výkonu |

> Údaje ověřte na webu programu.

` + "```go\nfmt.Println(\"ahoj\")\n```" + `

---
- [x] hotovo
`

func TestParse(t *testing.T) {
	bl := Parse(sample)
	kinds := []string{}
	for _, b := range bl {
		kinds = append(kinds, b.Kind)
	}
	got := strings.Join(kinds, " ")
	if got != "heading para heading list list table quote code hr list" {
		t.Fatalf("blocks: %s", got)
	}
	if bl[1].Spans[1].Text != "dotací" || !bl[1].Spans[1].Bold {
		t.Fatalf("bold span: %+v", bl[1].Spans)
	}
	if it := bl[3].Items; len(it) != 3 || it[2].Level != 1 {
		t.Fatalf("nested list: %+v", it)
	}
	if bl[5].Rows[1][0][0].Text != "Kotlíková dotace" {
		t.Fatalf("table: %+v", bl[5].Rows)
	}
}

func TestDOCX(t *testing.T) {
	var buf bytes.Buffer
	if err := DOCX(&buf, "Report", sample); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	var doc string
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			rc, _ := f.Open()
			b, _ := io.ReadAll(rc)
			doc = string(b)
		}
	}
	for _, want := range []string{`w:val="Heading1"`, "Tepelná čerpadla 2026", "<w:b/>", "<w:tbl>", "Kotlíková dotace", `w:val="Code"`, "(https://www.mzp.cz)"} {
		if !strings.Contains(doc, want) {
			t.Errorf("document.xml lacks %q", want)
		}
	}
}

func TestPDF(t *testing.T) {
	if _, err := findFonts(); err != nil {
		t.Skip(err)
	}
	var buf bytes.Buffer
	if err := PDF(&buf, "Report", sample+"\n"+strings.Repeat("Dlouhý odstavec s háčky a čárkami. ", 400)); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF-")) || buf.Len() < 5000 {
		t.Fatalf("not a PDF: %d bytes", buf.Len())
	}
}

// Models love emoji in reports. fpdf panicked on runes past U+FFFF, which
// left the Artifact call hanging; they are now dropped.
func TestPDFWithEmoji(t *testing.T) {
	var out bytes.Buffer
	md := "# 🏆 Nejlepší modely\n\n| Model | Kvalita |\n|---|---|\n| Qwen | ⭐⭐⭐⭐⭐ |\n\n## 🚀 Doporučení\n\n- 🥇 **Qwen** ✅\n- ⚠️ pozor\n"
	if err := PDF(&out, "🏆 Report", md); err != nil {
		t.Fatal(err)
	}
	if out.Len() < 1000 {
		t.Fatalf("PDF too small: %d bytes", out.Len())
	}
}
