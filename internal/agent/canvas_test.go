package agent

import (
	"strings"
	"testing"
)

func TestCanvasPage(t *testing.T) {
	p := CanvasPage("function draw(ctx,w,h,t){}\n// </script><b>", CanvasTheme{BG: "#000", FG: "#fff", Dim: "#888", Grid: "#333", Accent: "#0af"})
	if !strings.Contains(p, "function draw(ctx,w,h,t){}") || strings.Contains(p, "// </script><b>") || !strings.Contains(p, `<\/script><b>`) {
		t.Error("code not embedded safely")
	}
	if !strings.Contains(p, `"bg":"#000"`) || strings.Contains(p, "#FFFDF8") {
		t.Error("theme not applied")
	}
	if d := CanvasPage("x", CanvasTheme{}); !strings.Contains(d, "#FFFDF8") {
		t.Error("default theme missing")
	}
}
