package component

import (
	"bytes"
	"strings"
	"testing"
)

func TestCopyButtonEscapesValue(t *testing.T) {
	payloads := []string{
		"`${alert(1)}`",
		`postgres://u:p@h/db?x=</script><script>alert(1)</script>`,
		`" onmouseover="alert(1)`,
		`\`,
	}
	for _, p := range payloads {
		var buf bytes.Buffer
		if err := CopyButtonSm(p).Render(&buf); err != nil {
			t.Fatal(err)
		}
		out := buf.String()
		if strings.Contains(out, "<script") {
			t.Fatalf("script tag in output for %q: %s", p, out)
		}
		if strings.Contains(out, `" onmouseover="`) {
			t.Fatalf("attribute breakout for %q: %s", p, out)
		}
	}
}
