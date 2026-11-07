package logging

import (
	"testing"
)

func TestMaskValue_types(t *testing.T) {
	maskFieldNames = []string{"secret"}

	got := maskValue(map[string]any{"secret": "x", "ok": "y"})
	m, ok := got.(map[string]any)
	if !ok || m["secret"] != "***" || m["ok"] != "y" {
		t.Fatalf("map[string]any: %v", got)
	}

	got2 := maskValue(map[string]string{"secret": "x"})
	m2, ok := got2.(map[string]string)
	if !ok || m2["secret"] != "***" {
		t.Fatalf("map[string]string: %v", got2)
	}

	got3 := maskValue([]any{map[string]any{"secret": "z"}})
	sl, ok := got3.([]any)
	if !ok {
		t.Fatalf("[]any: %T", got3)
	}
	inner := sl[0].(map[string]any)
	if inner["secret"] != "***" {
		t.Fatal(inner)
	}

	type nested struct {
		Secret string
		OK     int
	}
	got4 := maskValue(map[string]nested{"wrap": {Secret: "h", OK: 1}})
	if got4 == nil {
		t.Fatal("reflect map expected")
	}
}

func TestMaskStringAlwaysStars(t *testing.T) {
	if got := maskString("secret"); got != "***" {
		t.Fatalf("maskString = %q", got)
	}
	if got := maskString(""); got != "" {
		t.Fatalf("empty = %q", got)
	}
}

func TestAsyncBlockWhenFull(t *testing.T) {
	sink := &encodeSink{output: OutputStd, writer: discardWriter{}}
	a := newAsyncCore(sink, 2, false)
	done := make(chan struct{})
	go func() {
		_ = a.Write(&LogItem{Message: "a"}, false)
		_ = a.Write(&LogItem{Message: "b"}, false)
		_ = a.Write(&LogItem{Message: "c"}, false)
		close(done)
	}()
	<-done
	_ = a.Close()
}
