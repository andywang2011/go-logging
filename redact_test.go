package logging

import (
	"net/http"
	"strings"
	"testing"
)

func TestDefaultRedactor_Headers(t *testing.T) {
	r := NewDefaultRedactor()
	h := http.Header{}
	h.Set("Authorization", "Bearer secret")
	h.Set("Authorization-Token", "tok")
	h.Set("Content-Type", "application/json")
	out := r.RedactHeaders(h)
	if out["Authorization"] != "***" || out["Authorization-Token"] != "***" {
		t.Fatalf("%v", out)
	}
	if out["Content-Type"] != "application/json" {
		t.Fatalf("%v", out)
	}
}

func TestDefaultRedactor_JSONBody(t *testing.T) {
	r := NewDefaultRedactor()
	in := `{"username":"a","password":"p","nested":{"token":"t","ok":1}}`
	out := r.RedactJSONBody(in)
	if strings.Contains(out, `"p"`) || strings.Contains(out, `"t"`) {
		t.Fatalf("not redacted: %s", out)
	}
	if !strings.Contains(out, `"username":"a"`) || !strings.Contains(out, `"ok":1`) {
		t.Fatalf("over-redacted: %s", out)
	}
}

func TestDefaultRedactor_NonJSONOmitted(t *testing.T) {
	r := NewDefaultRedactor()
	if got := r.RedactJSONBody("not-json"); got != "[non-json body omitted]" {
		t.Fatalf("%q", got)
	}
}

func TestDefaultRedactor_NonJSONVariants(t *testing.T) {
	r := NewDefaultRedactor()
	const omitted = "[non-json body omitted]"
	for _, in := range []string{"{broken", "[1,2", "  plain text  ", "password=hunter2"} {
		if got := r.RedactJSONBody(in); got != omitted {
			t.Errorf("RedactJSONBody(%q) = %q, want %q", in, got, omitted)
		}
	}
	if got := r.RedactJSONBody(""); got != "" {
		t.Errorf("empty body: %q", got)
	}
}
