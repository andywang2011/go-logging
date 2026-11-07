package logging

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func captureStdout(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func setupStdout(t *testing.T) {
	t.Helper()
	if err := SetupFromConfig(Config{Output: OutputStd, AESDisabled: true, AsyncSync: true}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = Close() })
}

func TestNewAndSetters(t *testing.T) {
	l := New()
	if l.RequestID != "" || l.V1 != "" {
		t.Fatal("New() should be empty")
	}
	l.SetRequestID("r1")
	l.Setv1("a")
	l.Setv2("b")
	l.Setv3("c")
	if l.RequestID != "r1" || l.V1 != "a" || l.V2 != "b" || l.V3 != "c" {
		t.Fatal("setters failed")
	}

	v := NewV("m", "api", "h")
	if v.V1 != "m" || v.V2 != "api" || v.V3 != "h" {
		t.Fatal("NewV mismatch")
	}

	lg := NewLogger("rid", "v1", "v2", "v3")
	if lg.RequestID != "rid" {
		t.Fatal("NewLogger mismatch")
	}
}

func TestConvert(t *testing.T) {
	if got := convert(errors.New("e")); got != "e" {
		t.Fatalf("error convert = %v", got)
	}
	if got := convert(stringerStub{s: "s"}); got != "s" {
		t.Fatalf("stringer convert = %v", got)
	}
	if got := convert([]byte("b")); got != "b" {
		t.Fatalf("bytes convert = %v", got)
	}
	if got := convert(42); got != 42 {
		t.Fatalf("default convert = %v", got)
	}
}

type stringerStub struct{ s string }

func (s stringerStub) String() string { return s.s }

func TestLoggerLevels(t *testing.T) {
	setupStdout(t)
	l := NewLogger("req-1", "v1", "v2", "v3")

	out := captureStdout(func() {
		l.Info("info-msg", map[string]string{"k": "v"}, map[string]int{"n": 1})
		l.Warn("warn-msg", nil, nil)
		l.ErrorL("err-msg", "req", "resp")
		l.Fatal("fatal-msg", nil, nil)
	})
	for _, sub := range []string{"info-msg", "warn-msg", "err-msg", "fatal-msg", "req-1"} {
		if !strings.Contains(out, sub) {
			t.Errorf("output missing %q: %s", sub, out)
		}
	}
}

func TestPackageLevelLoggers(t *testing.T) {
	setupStdout(t)
	out := captureStdout(func() {
		Info("rid", "j", "sync", "run", "pkg-info", nil, nil)
		Warn("rid", "j", "sync", "run", "pkg-warn", nil, nil)
		Error("rid", "j", "sync", "run", "pkg-err", nil, nil)
		Fatal("rid", "j", "sync", "run", "pkg-fatal", nil, nil)
	})
	for _, sub := range []string{"pkg-info", "pkg-warn", "pkg-err", "pkg-fatal"} {
		if !strings.Contains(out, sub) {
			t.Errorf("output missing %q", sub)
		}
	}
}

func TestStashAttrOption(t *testing.T) {
	setupStdout(t)
	start := time.Now().Add(-10 * time.Millisecond)
	l := New()

	out := captureStdout(func() {
		l.Info("opt", "req", "resp", AttrOption{
			RequestID:        "from-opt",
			StartTime:        &start,
			EncryptStatus:    EncryptStatusDisable,
			DisablePrintBody: true,
		})
	})
	if !strings.Contains(out, "from-opt") {
		t.Errorf("RequestID from opt: %s", out)
	}
	if !strings.Contains(out, `"response":null`) {
		t.Errorf("response should be null: %s", out)
	}
	if !strings.Contains(out, `"execute_time"`) {
		t.Errorf("execute_time missing: %s", out)
	}
}

func TestStashEncryptStatusEnable(t *testing.T) {
	err := SetupFromConfig(Config{
		Output:      OutputStd,
		AESDisabled: true,
		AESFields:   []string{"secret"},
		AsyncSync:   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer Close()
	out := captureStdout(func() {
		New().Info("enc", map[string]string{"secret": "plain"}, nil, AttrOption{
			EncryptStatus: EncryptStatusEnable,
		})
	})
	if strings.Contains(out, "plain") {
		t.Errorf("expected encrypted output, got: %s", out)
	}
}

func TestStashAutoV2Caller(t *testing.T) {
	setupStdout(t)
	out := captureStdout(func() {
		New().Info("auto-v2", nil, nil)
	})
	if !strings.Contains(out, "logging_test.go") {
		t.Errorf("expected caller file in v2, got: %s", out)
	}
}

func TestSetupFromConfig_RejectsAESKeySizes(t *testing.T) {
	for _, size := range []int{16, 24, 32} {
		key := strings.Repeat("k", size)
		err := SetupFromConfig(Config{
			Output:      OutputStd,
			AESKey:      key,
			AESFields:   []string{"field"},
			AESDisabled: false,
		})
		if !errors.Is(err, ErrAESKeyUnsupported) {
			t.Errorf("key size %d: err = %v, want ErrAESKeyUnsupported", size, err)
		}
	}
}

func TestSetupFromConfig_FileOutput(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	if err := SetupFromConfig(Config{
		Output:          OutputFile,
		LogTopic:        "ut",
		FileMaxSize:     1,
		FileBackupCount: 1,
		AESDisabled:     true,
		AsyncSync:       true,
	}); err != nil {
		t.Fatal(err)
	}
	defer Close()
	New().Info("file-log", nil, nil)
	path := "log/ut/app.log"
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("log file not created: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "file-log") {
		t.Fatalf("log content: %s", data)
	}
}

func TestSetupFromConfig_EmptyLogTopicDefaultsApp(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	_ = os.Chdir(dir)
	t.Cleanup(func() { _ = os.Chdir(wd) })

	if err := SetupFromConfig(Config{
		Output:      OutputFile,
		LogTopic:    "",
		AESDisabled: true,
		AsyncSync:   true,
	}); err != nil {
		t.Fatal(err)
	}
	defer Close()
	New().Info("default-topic", nil, nil)
	if _, err := os.Stat("log/app/app.log"); err != nil {
		t.Fatal(err)
	}
}

func TestOutputFmt(t *testing.T) {
	if err := SetupFromConfig(Config{Output: OutputFmt, AESDisabled: true}); err != nil {
		t.Fatal(err)
	}
	_ = captureStdout(func() {
		New().Info("fmt-mode", nil, nil)
	})
}

func TestOutputFmt_withEncrypt(t *testing.T) {
	if err := SetupFromConfig(Config{
		Output:      OutputFmt,
		AESDisabled: false,
		AESFields:   []string{"token"},
	}); err != nil {
		t.Fatal(err)
	}
	_ = captureStdout(func() {
		New().Info("fmt-enc", map[string]string{"token": "hide-me"}, nil)
	})
}

func TestGetFilterCallers(t *testing.T) {
	file, line, ok := getFilterCallers()
	if !ok || file == "" || line <= 0 {
		t.Fatalf("getFilterCallers = %s:%d ok=%v", file, line, ok)
	}
}
