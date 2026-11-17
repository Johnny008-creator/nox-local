package docexport

import (
	"archive/zip"
	"fmt"
	"io"
	"strings"
	"time"
)

// DOCX writes the markdown document as a Word file: real heading styles
// (so Word's navigation pane and table of contents work), lists, tables,
// code blocks and inline formatting.
func DOCX(w io.Writer, title, markdown string) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("DOCX export failed: %v", p)
		}
	}()
	var body strings.Builder
	if title != "" {
		para(&body, "Title", []Span{{Text: title}})
	}
	for _, b := range Parse(markdown) {
		writeBlock(&body, b)
	}
	zw := zip.NewWriter(w)
	files := []struct{ name, data string }{
		{"[Content_Types].xml", contentTypes},
		{"_rels/.rels", rootRels},
		{"word/_rels/document.xml.rels", docRels},
		{"word/styles.xml", styles},
		{"docProps/core.xml", fmt.Sprintf(coreProps, esc(title), time.Now().UTC().Format(time.RFC3339))},
		{"word/document.xml", docHead + body.String() + docTail},
	}
	for _, f := range files {
		fw, err := zw.Create(f.name)
		if err != nil {
			return err
		}
		if _, err := io.WriteString(fw, f.data); err != nil {
			return err
		}
	}
	return zw.Close()
}

func writeBlock(b *strings.Builder, bl Block) {
	switch bl.Kind {
	case "heading":
		lvl := min(max(bl.Level, 1), 4)
		para(b, fmt.Sprintf("Heading%d", lvl), bl.Spans)
	case "para":
		para(b, "", bl.Spans)
	case "quote":
		para(b, "Quote", bl.Spans)
	case "code":
		for _, l := range strings.Split(bl.Code, "\n") {
			para(b, "Code", []Span{{Text: l, Code: true}})
		}
	case "hr":
		b.WriteString(`<w:p><w:pPr><w:pBdr><w:bottom w:val="single" w:sz="6" w:space="1" w:color="BFBFBF"/></w:pBdr></w:pPr></w:p>`)
	case "list":
		for _, it := range bl.Items {
			mark := "•"
			if it.Ordered {
				mark = fmt.Sprintf("%d.", it.Num)
			}
			indent := 360 + it.Level*360
			fmt.Fprintf(b, `<w:p><w:pPr><w:pStyle w:val="ListParagraph"/><w:ind w:left="%d" w:hanging="360"/></w:pPr>`, indent)
			run(b, Span{Text: mark + "\t"})
			for _, s := range it.Spans {
				run(b, s)
			}
			b.WriteString(`</w:p>`)
		}
	case "table":
		if len(bl.Rows) == 0 {
			return
		}
		b.WriteString(`<w:tbl><w:tblPr><w:tblStyle w:val="TableGrid"/><w:tblW w:w="5000" w:type="pct"/><w:tblBorders>`)
		for _, side := range []string{"top", "left", "bottom", "right", "insideH", "insideV"} {
			fmt.Fprintf(b, `<w:%s w:val="single" w:sz="4" w:space="0" w:color="BFBFBF"/>`, side)
		}
		b.WriteString(`</w:tblBorders></w:tblPr>`)
		for i, row := range bl.Rows {
			b.WriteString(`<w:tr>`)
			for _, cell := range row {
				b.WriteString(`<w:tc><w:tcPr><w:tcW w:w="0" w:type="auto"/>`)
				if i == 0 {
					b.WriteString(`<w:shd w:val="clear" w:color="auto" w:fill="F2F2F2"/>`)
				}
				b.WriteString(`</w:tcPr><w:p>`)
				for _, s := range cell {
					if i == 0 {
						s.Bold = true
					}
					run(b, s)
				}
				b.WriteString(`</w:p></w:tc>`)
			}
			b.WriteString(`</w:tr>`)
		}
		b.WriteString(`</w:tbl><w:p/>`)
	}
}

func para(b *strings.Builder, style string, spans []Span) {
	b.WriteString(`<w:p>`)
	if style != "" {
		fmt.Fprintf(b, `<w:pPr><w:pStyle w:val="%s"/></w:pPr>`, style)
	}
	for _, s := range spans {
		run(b, s)
	}
	b.WriteString(`</w:p>`)
}

func run(b *strings.Builder, s Span) {
	parts := strings.Split(s.Text, "\n")
	for i, part := range parts {
		b.WriteString(`<w:r><w:rPr>`)
		if s.Bold {
			b.WriteString(`<w:b/>`)
		}
		if s.Italic {
			b.WriteString(`<w:i/>`)
		}
		if s.Code {
			b.WriteString(`<w:rFonts w:ascii="Consolas" w:hAnsi="Consolas" w:cs="Consolas"/><w:shd w:val="clear" w:color="auto" w:fill="F2F2F2"/>`)
		}
		if s.Link != "" {
			b.WriteString(`<w:color w:val="1F5FAD"/><w:u w:val="single"/>`)
		}
		b.WriteString(`</w:rPr>`)
		if i > 0 {
			b.WriteString(`<w:br/>`)
		}
		pieces := strings.Split(part, "\t")
		for j, p := range pieces {
			if j > 0 {
				b.WriteString(`<w:tab/>`)
			}
			if p != "" {
				fmt.Fprintf(b, `<w:t xml:space="preserve">%s</w:t>`, esc(p))
			}
		}
		b.WriteString(`</w:r>`)
	}
	if s.Link != "" && !strings.Contains(s.Text, s.Link) && strings.HasPrefix(s.Link, "http") {
		fmt.Fprintf(b, `<w:r><w:rPr><w:color w:val="7F7F7F"/><w:sz w:val="18"/></w:rPr><w:t xml:space="preserve"> (%s)</w:t></w:r>`, esc(s.Link))
	}
}

var xmlEsc = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")

// esc escapes text for XML and drops control characters Word refuses.
func esc(s string) string {
	s = strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' {
			return -1
		}
		return r
	}, s)
	return xmlEsc.Replace(s)
}

const contentTypes = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
<Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
<Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/>
</Types>`

const rootRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/>
</Relationships>`

const docRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
</Relationships>`

const coreProps = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:dcterms="http://purl.org/dc/terms/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
<dc:title>%s</dc:title><dc:creator>NØX-Local</dc:creator>
<dcterms:created xsi:type="dcterms:W3CDTF">%s</dcterms:created>
</cp:coreProperties>`

const docHead = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>`

const docTail = `<w:sectPr><w:pgSz w:w="11906" w:h="16838"/><w:pgMar w:top="1134" w:right="1134" w:bottom="1134" w:left="1134" w:header="708" w:footer="708" w:gutter="0"/></w:sectPr></w:body></w:document>`

const styles = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:docDefaults><w:rPrDefault><w:rPr><w:rFonts w:ascii="Calibri" w:hAnsi="Calibri" w:cs="Calibri"/><w:sz w:val="22"/><w:lang w:val="cs-CZ"/></w:rPr></w:rPrDefault>
<w:pPrDefault><w:pPr><w:spacing w:after="120" w:line="276" w:lineRule="auto"/></w:pPr></w:pPrDefault></w:docDefaults>
<w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/></w:style>
<w:style w:type="paragraph" w:styleId="Title"><w:name w:val="Title"/><w:basedOn w:val="Normal"/><w:pPr><w:spacing w:after="240"/></w:pPr><w:rPr><w:b/><w:sz w:val="44"/><w:color w:val="1F2937"/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="heading 1"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:pPr><w:keepNext/><w:spacing w:before="360" w:after="120"/><w:outlineLvl w:val="0"/></w:pPr><w:rPr><w:b/><w:sz w:val="32"/><w:color w:val="1F2937"/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="Heading2"><w:name w:val="heading 2"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:pPr><w:keepNext/><w:spacing w:before="280" w:after="100"/><w:outlineLvl w:val="1"/></w:pPr><w:rPr><w:b/><w:sz w:val="27"/><w:color w:val="1F2937"/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="Heading3"><w:name w:val="heading 3"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:pPr><w:keepNext/><w:spacing w:before="220" w:after="80"/><w:outlineLvl w:val="2"/></w:pPr><w:rPr><w:b/><w:sz w:val="24"/><w:color w:val="374151"/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="Heading4"><w:name w:val="heading 4"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:pPr><w:keepNext/><w:outlineLvl w:val="3"/></w:pPr><w:rPr><w:b/><w:i/><w:color w:val="374151"/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="Quote"><w:name w:val="Quote"/><w:basedOn w:val="Normal"/><w:pPr><w:ind w:left="567"/><w:pBdr><w:left w:val="single" w:sz="12" w:space="8" w:color="BFBFBF"/></w:pBdr></w:pPr><w:rPr><w:i/><w:color w:val="595959"/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="Code"><w:name w:val="Code"/><w:basedOn w:val="Normal"/><w:pPr><w:spacing w:after="0" w:line="240" w:lineRule="auto"/><w:shd w:val="clear" w:color="auto" w:fill="F2F2F2"/><w:ind w:left="284"/></w:pPr><w:rPr><w:rFonts w:ascii="Consolas" w:hAnsi="Consolas" w:cs="Consolas"/><w:sz w:val="19"/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="ListParagraph"><w:name w:val="List Paragraph"/><w:basedOn w:val="Normal"/><w:pPr><w:spacing w:after="60"/><w:tabs><w:tab w:val="left" w:pos="360"/></w:tabs></w:pPr></w:style>
<w:style w:type="table" w:styleId="TableGrid"><w:name w:val="Table Grid"/><w:tblPr><w:tblCellMar><w:top w:w="60" w:type="dxa"/><w:left w:w="100" w:type="dxa"/><w:bottom w:w="60" w:type="dxa"/><w:right w:w="100" w:type="dxa"/></w:tblCellMar></w:tblPr></w:style>
</w:styles>`
