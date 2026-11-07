package logging

import (
	"bytes"
	"context"
	"strconv"
)

const (
	// XRequestID is the context key for the request correlation id.
	XRequestID = "X-Request-Id"

	compressedDescWordLen = 48
)

type __ctx__ struct {
	m map[string]string
	context.Context
}

func (c *__ctx__) Value(key any) any {
	ks, _ := key.(string)
	if v, ok := c.m[ks]; ok {
		return v
	}
	return c.Context.Value(key)
}

// GetContext returns a context carrying this logger's dimensions for downstream calls.
func (l *Logger) GetContext() context.Context {
	return &__ctx__{
		map[string]string{
			XRequestID: l.RequestID,
			"v1":       l.V1,
			"v2":       l.V2,
			"v3":       l.V3,
		},
		context.Background(),
	}
}

// NewContext returns a context with request_id and v1–v3 dimensions.
func NewContext(reqid, v1, v2, v3 string) context.Context {
	return &__ctx__{
		map[string]string{
			XRequestID: reqid,
			"v1":       v1,
			"v2":       v2,
			"v3":       v3,
		},
		context.Background(),
	}
}

// FromContext builds a Logger from context values set by NewContext or GetContext.
func FromContext(ctx context.Context) *Logger {
	l := &Logger{}
	l.RequestID, _ = ctx.Value(XRequestID).(string)
	l.V1, _ = ctx.Value("v1").(string)
	l.V2, _ = ctx.Value("v2").(string)
	l.V3, _ = ctx.Value("v3").(string)
	return l
}

// SpreadMaps flattens url.Values or http.Header slice values into map[string]any.
func SpreadMaps(m map[string][]string) map[string]any {
	res := make(map[string]any, len(m))
	for k, v := range m {
		if len(v) == 1 {
			res[k] = v[0]
		} else {
			res[k] = v
		}
	}
	return res
}

// Compress truncates b to keepLen bytes and inserts an omission marker for the middle section.
func Compress(b []byte, keepLen int) string {
	if l := len(b); l > keepLen+compressedDescWordLen {
		buf := bytes.NewBuffer(nil)
		buf.Grow(keepLen + compressedDescWordLen)
		buf.Write(b[:keepLen/2])
		buf.WriteString("***omitted{")
		buf.WriteString(strconv.Itoa(l - keepLen))
		buf.WriteString("}chars***")
		buf.Write(b[l-keepLen/2:])
		return buf.String()
	}
	return string(b)
}
