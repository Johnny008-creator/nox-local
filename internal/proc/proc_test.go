package proc

import "testing"

func TestShimPicksNodeModulesTarget(t *testing.T) {
	shim := `IF EXIST "%dp0%\node.exe" (
  SET "_prog=%dp0%\node.exe"
)
endLocal & goto #_undefined_# 2>NUL || title %COMSPEC% & "%_prog%"  "%dp0%\node_modules\@openai\codex\bin\codex.js" %*`
	var rel string
	for _, m := range shimTarget.FindAllSubmatch([]byte(shim), -1) {
		if string(m[1]) != `node.exe` {
			rel = string(m[1])
		}
	}
	if rel != `node_modules\@openai\codex\bin\codex.js` {
		t.Fatalf("got %q", rel)
	}
}
