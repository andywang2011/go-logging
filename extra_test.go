package logging

import (
	"context"
	"strings"
	"testing"
)

func TestNewContextAndFromContext(t *testing.T) {
	ctx := NewContext("rid", "v1", "v2", "v3")
	l := FromContext(ctx)
	if l.RequestID != "rid" || l.V1 != "v1" || l.V2 != "v2" || l.V3 != "v3" {
		t.Fatalf("FromContext = %+v", l)
	}

	l2 := NewLogger("a", "b", "c", "d")
	ctx2 := l2.GetContext()
	l3 := FromContext(ctx2)
	if l3.RequestID != "a" || l3.V1 != "b" {
		t.Fatalf("GetContext roundtrip = %+v", l3)
	}
}

func TestFromContextEmpty(t *testing.T) {
	l := FromContext(NewContext("", "", "", ""))
	if l.RequestID != "" {
		t.Fatal("expected empty logger")
	}
}

func TestAttach_PreservesParentCancelAndReadsDimensions(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx := Attach(parent, "rid-1", "/api/ping", "", "")
	l := FromContext(ctx)
	if l.RequestID != "rid-1" || l.V1 != "/api/ping" {
		t.Fatalf("FromContext = %+v", l)
	}
	cancel()
	select {
	case <-ctx.Done():
	default:
		t.Fatal("expected attached ctx to inherit parent cancel")
	}
}

func TestAttach_EmptyFieldsDoNotOverwrite(t *testing.T) {
	base := Attach(context.Background(), "keep-id", "keep-v1", "keep-v2", "keep-v3")
	ctx := Attach(base, "", "", "new-v2", "")
	l := FromContext(ctx)
	if l.RequestID != "keep-id" || l.V1 != "keep-v1" || l.V2 != "new-v2" || l.V3 != "keep-v3" {
		t.Fatalf("FromContext = %+v", l)
	}
}

func TestAttach_NilParent(t *testing.T) {
	ctx := Attach(nil, "rid", "v1", "", "")
	l := FromContext(ctx)
	if l.RequestID != "rid" || l.V1 != "v1" {
		t.Fatalf("FromContext = %+v", l)
	}
}

func TestSpreadMaps(t *testing.T) {
	m := map[string][]string{
		"single": {"one"},
		"multi":  {"a", "b"},
	}
	got := SpreadMaps(m)
	if got["single"] != "one" {
		t.Fatalf("single = %v", got["single"])
	}
	sl, ok := got["multi"].([]string)
	if !ok || len(sl) != 2 {
		t.Fatalf("multi = %v", got["multi"])
	}
}

func TestCompress_short(t *testing.T) {
	b := []byte("short")
	if got := Compress(b, 100); got != "short" {
		t.Fatalf("got %q", got)
	}
}

func TestCompress_long(t *testing.T) {
	b := []byte(strings.Repeat("x", 200))
	got := Compress(b, 20)
	if !strings.Contains(got, "omitted") {
		t.Fatalf("expected truncation marker: %q", got)
	}
	if len(got) >= len(b) {
		t.Fatal("expected compressed length")
	}
}

func TestCtxValue_nonStringKey(t *testing.T) {
	type ctxKey int
	var k ctxKey = 1
	base := context.WithValue(context.Background(), k, "val")
	c := &__ctx__{m: map[string]string{"v1": "a"}, Context: base}
	if got := c.Value(k); got != "val" {
		t.Fatalf("non-string key = %v", got)
	}
	if got := c.Value("v1"); got != "a" {
		t.Fatalf("map key = %v", got)
	}
}
