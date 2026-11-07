package logging

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Redactor redacts sensitive HTTP headers and JSON/map payloads for logs and audit trails.
type Redactor interface {
	RedactHeaders(h http.Header) map[string]any
	RedactJSONBody(body string) string
	RedactMap(m map[string]any) map[string]any
}

// DefaultSensitiveHeaders are redacted regardless of AES field config.
var DefaultSensitiveHeaders = []string{
	"Authorization",
	"Authorization-Token",
	"Authorization-AppKey",
	"Cookie",
	"Set-Cookie",
	"X-Api-Key",
	"X-Auth-Token",
	"Proxy-Authorization",
}

// DefaultSensitiveFields are matched case-insensitively as substrings of JSON keys.
var DefaultSensitiveFields = []string{
	"password",
	"passwd",
	"secret",
	"token",
	"access_key",
	"private_key",
	"card",
	"ssn",
	"cvv",
	"pin",
	"id_card",
	"id_number",
	"passport",
	"birthday",
	"real_name",
	"email",
	"phone",
	"mobile",
	"address",
	"totp_secret",
	"recovery_code",
}

const redactedPlaceholder = "***"

// DefaultRedactor implements Redactor with configurable header and field name lists.
type DefaultRedactor struct {
	Headers []string
	Fields  []string
}

// NewDefaultRedactor returns a Redactor with DefaultSensitiveHeaders and DefaultSensitiveFields.
func NewDefaultRedactor() *DefaultRedactor {
	return &DefaultRedactor{
		Headers: append([]string(nil), DefaultSensitiveHeaders...),
		Fields:  append([]string(nil), DefaultSensitiveFields...),
	}
}

// RedactHeaders copies headers, replacing sensitive values with ***.
func (r *DefaultRedactor) RedactHeaders(h http.Header) map[string]any {
	out := make(map[string]any, len(h))
	if h == nil {
		return out
	}
	for k, vals := range h {
		if r.headerSensitive(k) {
			out[k] = redactedPlaceholder
			continue
		}
		if len(vals) == 1 {
			out[k] = vals[0]
		} else {
			cp := make([]string, len(vals))
			copy(cp, vals)
			out[k] = cp
		}
	}
	return out
}

// RedactJSONBody parses JSON object/array payloads and redacts sensitive keys.
// Non-JSON bodies are replaced with "[non-json body omitted]" (GDPR data minimization).
func (r *DefaultRedactor) RedactJSONBody(body string) string {
	const omitted = "[non-json body omitted]"
	if body == "" {
		return body
	}
	trim := strings.TrimSpace(body)
	if len(trim) == 0 || (trim[0] != '{' && trim[0] != '[') {
		return omitted
	}
	var v any
	if err := json.Unmarshal([]byte(body), &v); err != nil {
		return omitted
	}
	redacted := r.redactValue(v)
	b, err := json.Marshal(redacted)
	if err != nil {
		return omitted
	}
	return string(b)
}

// RedactMap returns a deep copy with sensitive keys replaced.
func (r *DefaultRedactor) RedactMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out, _ := r.redactValue(m).(map[string]any)
	return out
}

func (r *DefaultRedactor) headerSensitive(name string) bool {
	nl := strings.ToLower(name)
	for _, h := range r.Headers {
		if strings.EqualFold(h, name) || strings.HasPrefix(nl, strings.ToLower(h)) {
			return true
		}
	}
	return false
}

func (r *DefaultRedactor) fieldSensitive(key string) bool {
	kl := strings.ToLower(key)
	for _, f := range r.Fields {
		if f == "" {
			continue
		}
		if strings.Contains(kl, strings.ToLower(f)) {
			return true
		}
	}
	return false
}

func (r *DefaultRedactor) redactValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, child := range val {
			if r.fieldSensitive(k) {
				out[k] = redactedPlaceholder
			} else {
				out[k] = r.redactValue(child)
			}
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, child := range val {
			out[i] = r.redactValue(child)
		}
		return out
	default:
		return v
	}
}
