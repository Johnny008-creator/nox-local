package agent

import (
	_ "embed"
	"encoding/json"
	"strings"
)

//go:embed canvas.html
var canvasPage string

// CanvasTheme is the colours a canvas artifact draws with.
type CanvasTheme struct {
	BG     string `json:"bg"`
	FG     string `json:"fg"`
	Dim    string `json:"dim"`
	Grid   string `json:"grid"`
	Accent string `json:"accent"`
}

// CanvasPage wraps a canvas artifact's JavaScript in the runtime page:
// a full-size canvas, the animation loop, nox.* chart helpers and an error
// overlay. A zero theme keeps the light default.
func CanvasPage(code string, theme CanvasTheme) string {
	page := canvasPage
	if theme.BG != "" {
		b, _ := json.Marshal(theme)
		start, end := strings.Index(page, "/*THEME*/"), strings.Index(page, "/*END*/")
		page = page[:start+len("/*THEME*/")] + string(b) + page[end:]
	}
	// The code sits inside a <script>; a literal </script> would end it early.
	code = strings.ReplaceAll(code, "</script", `<\/script`)
	return strings.Replace(page, "/*CODE*/", code, 1)
}
