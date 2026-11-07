package logging

import (
	"reflect"
	"strings"
)

// maskLogItem applies field-level masking to request/response before JSON encoding.
// Matched string values are always replaced with "***" (AES masking was removed).
func maskLogItem(item *LogItem) {
	cryptoMu.RLock()
	defer cryptoMu.RUnlock()
	if item == nil || len(maskFieldNames) == 0 {
		return
	}
	item.Request = maskValue(item.Request)
	item.Response = maskValue(item.Response)
}

func maskValue(v any) any {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case map[string]any:
		return maskMapStringAny(val)
	case map[string]string:
		out := make(map[string]string, len(val))
		for k, s := range val {
			if fieldMatches(k) {
				out[k] = maskString(s)
			} else {
				out[k] = s
			}
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, el := range val {
			out[i] = maskValue(el)
		}
		return out
	case []string:
		return val
	default:
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Map {
			return maskReflectMap(rv)
		}
		if rv.Kind() == reflect.Slice && rv.Type().Elem().Kind() != reflect.Uint8 {
			return maskReflectSlice(rv)
		}
		return v
	}
}

func maskMapStringAny(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, val := range m {
		if fieldMatches(k) {
			out[k] = maskScalar(val)
		} else {
			out[k] = maskValue(val)
		}
	}
	return out
}

func maskReflectMap(rv reflect.Value) map[string]any {
	out := make(map[string]any)
	for _, key := range rv.MapKeys() {
		ks := key.String()
		val := rv.MapIndex(key).Interface()
		if fieldMatches(ks) {
			out[ks] = maskScalar(val)
		} else {
			out[ks] = maskValue(val)
		}
	}
	return out
}

func maskReflectSlice(rv reflect.Value) []any {
	n := rv.Len()
	out := make([]any, n)
	for i := 0; i < n; i++ {
		out[i] = maskValue(rv.Index(i).Interface())
	}
	return out
}

func maskScalar(v any) any {
	switch s := v.(type) {
	case string:
		return maskString(s)
	case []byte:
		return maskString(string(s))
	default:
		return "***"
	}
}

func maskString(s string) string {
	if s == "" {
		return ""
	}
	return "***"
}

func fieldMatches(key string) bool {
	kl := strings.ToLower(key)
	for _, f := range maskFieldNames {
		if f == "" {
			continue
		}
		if strings.Contains(kl, f) {
			return true
		}
	}
	return false
}
