package logging

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	validMaskFieldName = regexp.MustCompile(`^\w+$`)
	cryptoMu           sync.RWMutex
)

// ErrAESKeyUnsupported is returned when Config.AESKey is non-empty.
// AES field masking was removed; matched fields are always replaced with "***".
var ErrAESKeyUnsupported = errors.New("logging: AESKey is no longer supported; use AESFields with *** masking only")

type level int8

const (
	_ level = iota
	FatalLevel
	ErrorLevel
	WarnLevel
	InfoLevel
)

const timeFormat = "2006-01-02 15:04:05.000000"

const (
	// EncryptStatusDefault uses the global masking configuration.
	EncryptStatusDefault = 0
	// EncryptStatusEnable forces field masking for this log line.
	EncryptStatusEnable = 1
	// EncryptStatusDisable disables field masking for this log line.
	EncryptStatusDisable = 2
)

var (
	// IsSetup is true after a successful SetupFromConfig call.
	IsSetup        bool
	aesDisabled    bool
	maskFieldNames []string // lowercase field names for substring match
)

// Logger carries request-scoped dimensions written into each log line.
type Logger struct {
	// RequestID is the correlation id (JSON request_id).
	RequestID string
	// V1 is an optional business dimension (JSON v1).
	V1 string
	// V2 is an optional business dimension (JSON v2); auto-filled with caller file:line when empty.
	V2 string
	// V3 is an optional business dimension (JSON v3).
	V3 string
}

// LogItem is the JSON payload written for each log line.
type LogItem struct {
	// Level is the numeric severity (see Level* constants).
	Level level `json:"level"`
	// RequestID is the correlation id.
	RequestID string `json:"request_id"`
	// V1 is the first optional business dimension.
	V1 string `json:"v1"`
	// V2 is the second optional business dimension.
	V2 string `json:"v2"`
	// V3 is the third optional business dimension.
	V3 string `json:"v3"`
	// Message is the human-readable log message.
	Message string `json:"message"`
	// Request is the request payload (any JSON-serializable value).
	Request any `json:"request"`
	// Response is the response payload (any JSON-serializable value).
	Response any `json:"response"`
	// CreateTime is the timestamp string (microsecond precision).
	CreateTime string `json:"create_time"`
	// ExecuteTime is elapsed milliseconds since AttrOption.StartTime when set.
	ExecuteTime int64 `json:"execute_time"`
}

// AttrOption overrides per-line logging behavior.
type AttrOption struct {
	// RequestID overrides the logger request id when non-empty.
	RequestID string `json:"request_id"`
	// StartTime enables execute_time calculation from this instant.
	StartTime *time.Time `json:"start_time"`
	// EncryptStatus overrides global masking (EncryptStatus* constants).
	EncryptStatus int `json:"encrypt_status"`
	// DisablePrintBody clears Response before encoding when true.
	DisablePrintBody bool `json:"disable_print_body"`
}

// SetupFromConfig initializes the logging pipeline from cfg; call once at process startup.
func SetupFromConfig(cfg Config) error {
	coreMu.Lock()
	defer coreMu.Unlock()
	_ = closeCoreLocked()
	if err := configureCrypto(cfg); err != nil {
		return err
	}
	aesDisabled = cfg.AESDisabled
	core, err := buildCore(cfg)
	if err != nil {
		return err
	}
	defaultCore = core
	IsSetup = true
	return nil
}

func configureCrypto(cfg Config) error {
	cryptoMu.Lock()
	defer cryptoMu.Unlock()
	maskFieldNames = nil

	if cfg.AESKey != "" {
		return ErrAESKeyUnsupported
	}

	for _, f := range cfg.AESFields {
		if !validMaskFieldName.MatchString(f) {
			return fmt.Errorf("invalid mask field name: %s", f)
		}
		maskFieldNames = append(maskFieldNames, strings.ToLower(f))
	}
	return nil
}

// New returns a Logger with empty dimensions.
func New() *Logger {
	return &Logger{}
}

// NewV returns a Logger with v1–v3 dimensions preset.
func NewV(v1, v2, v3 string) *Logger {
	return &Logger{
		V1: v1,
		V2: v2,
		V3: v3,
	}
}

// NewLogger returns a Logger with all dimensions preset.
func NewLogger(reqid, v1, v2, v3 string) *Logger {
	return &Logger{
		RequestID: reqid,
		V1:        v1,
		V2:        v2,
		V3:        v3,
	}
}

// SetRequestID sets the request correlation id.
func (l *Logger) SetRequestID(requestID string) {
	l.RequestID = requestID
}

// Setv1 sets the v1 dimension.
func (l *Logger) Setv1(v1 string) {
	l.V1 = v1
}

// Setv2 sets the v2 dimension.
func (l *Logger) Setv2(v2 string) {
	l.V2 = v2
}

// Setv3 sets the v3 dimension.
func (l *Logger) Setv3(v3 string) {
	l.V3 = v3
}

func convert(val any) any {
	switch v := val.(type) {
	case error:
		return v.Error()
	case fmt.Stringer:
		return v.String()
	case []byte:
		return string(v)
	default:
		return v
	}
}

func (l *Logger) stash(lev level, msg string, req, resp any, opts ...AttrOption) {
	logs := &LogItem{
		Level:      lev,
		RequestID:  l.RequestID,
		V1:         l.V1,
		V2:         l.V2,
		V3:         l.V3,
		Message:    msg,
		Request:    convert(req),
		Response:   convert(resp),
		CreateTime: time.Now().Format(timeFormat),
	}
	if logs.V2 == "" {
		if file, line, ok := getFilterCallers(); ok {
			logs.V2 = file + ":" + strconv.Itoa(line)
		}
	}

	cryptoMu.RLock()
	encrypt := !aesDisabled
	cryptoMu.RUnlock()
	if len(opts) > 0 {
		opt := opts[0]
		if logs.RequestID == "" {
			logs.RequestID = opt.RequestID
		}
		if opt.StartTime != nil {
			logs.ExecuteTime = time.Since(*opt.StartTime).Milliseconds()
		}

		// Per-line EncryptStatus overrides the global masking flag.
		if opt.EncryptStatus == EncryptStatusEnable {
			encrypt = true
		} else if opt.EncryptStatus == EncryptStatusDisable {
			encrypt = false
		}
		if opt.DisablePrintBody {
			logs.Response = nil
		}
	}
	writeLog(logs, encrypt)
}

// Info writes an info-level log line.
func (l *Logger) Info(message string, request, response any, opts ...AttrOption) {
	l.stash(InfoLevel, message, request, response, opts...)
}

// Warn writes a warn-level log line.
func (l *Logger) Warn(message string, request, response any, opts ...AttrOption) {
	l.stash(WarnLevel, message, request, response, opts...)
}

// ErrorL writes an error-level log line.
func (l *Logger) ErrorL(message string, request, response any, opts ...AttrOption) {
	l.stash(ErrorLevel, message, request, response, opts...)
}

// Fatal writes a fatal-level log line without calling os.Exit; the caller must exit the process.
func (l *Logger) Fatal(message string, request, response any, opts ...AttrOption) {
	l.stash(FatalLevel, message, request, response, opts...)
}

// Info writes an info-level log line with explicit dimensions.
func Info(reqID, v1, v2, v3, message string, request, response any, opts ...AttrOption) {
	NewLogger(reqID, v1, v2, v3).stash(InfoLevel, message, request, response, opts...)
}

// Warn writes a warn-level log line with explicit dimensions.
func Warn(reqID, v1, v2, v3, message string, request, response any, opts ...AttrOption) {
	NewLogger(reqID, v1, v2, v3).stash(WarnLevel, message, request, response, opts...)
}

// Error writes an error-level log line with explicit dimensions.
func Error(reqID, v1, v2, v3, message string, request, response any, opts ...AttrOption) {
	NewLogger(reqID, v1, v2, v3).stash(ErrorLevel, message, request, response, opts...)
}

// Fatal writes a fatal-level log line without calling os.Exit.
func Fatal(reqID, v1, v2, v3, message string, request, response any, opts ...AttrOption) {
	NewLogger(reqID, v1, v2, v3).stash(FatalLevel, message, request, response, opts...)
}

// getFilterCallers returns caller file:line when v2 is unset (skips this package, /mod/, stdlib).
func getFilterCallers() (file string, line int, ok bool) {
	var pcs [16]uintptr
	n := runtime.Callers(2, pcs[:])
	if n == 0 {
		return "", 0, false
	}
	frames := runtime.CallersFrames(pcs[:n])
	const maxCallerScan = len(pcs)
	for i := 0; i < maxCallerScan; i++ {
		frame, more := frames.Next()
		if !skipCallerFrame(frame) {
			return frame.File, frame.Line, true
		}
		if !more {
			break
		}
	}
	return "", 0, false
}

// skipCallerFrame reports whether a stack frame should be skipped when resolving v2.
func skipCallerFrame(frame runtime.Frame) bool {
	if strings.Contains(frame.File, "/mod/") {
		return true
	}
	if strings.Contains(frame.File, "go-logging"+string(os.PathSeparator)) {
		return !strings.HasSuffix(frame.File, "_test.go")
	}
	fn := frame.Function
	for _, prefix := range []string{
		"runtime.", "internal/", "testing.", "reflect.",
	} {
		if strings.HasPrefix(fn, prefix) {
			return true
		}
	}
	return false
}
