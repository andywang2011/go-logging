package logging

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func syncConfig(cfg Config) Config {
	cfg.AsyncSync = true
	return cfg
}

func TestEncodeSinkStdout(t *testing.T) {
	if err := SetupFromConfig(syncConfig(Config{Output: OutputStd, AESDisabled: true})); err != nil {
		t.Fatal(err)
	}
	defer Close()

	out := captureStdout(func() {
		New().Info("stdout-handle", nil, nil)
	})
	if !strings.Contains(out, "stdout-handle") {
		t.Fatalf("got %s", out)
	}
}

func TestEncodeSinkDefaultStdout(t *testing.T) {
	if err := SetupFromConfig(syncConfig(Config{Output: "unknown", AESDisabled: true})); err != nil {
		t.Fatal(err)
	}
	defer Close()

	out := captureStdout(func() {
		New().Info("default-out", nil, nil)
	})
	if !strings.Contains(out, "default-out") {
		t.Fatal("default output should use stdout")
	}
}

func TestEncodeSinkFileMask(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	_ = os.Chdir(dir)
	t.Cleanup(func() { _ = os.Chdir(wd) })

	if err := SetupFromConfig(syncConfig(Config{
		Output:      OutputFile,
		LogTopic:    "enc",
		AESDisabled: false,
		AESFields:   []string{"token"},
	})); err != nil {
		t.Fatal(err)
	}
	defer Close()

	New().Info("enc-file", map[string]string{"token": "abc"}, nil)
	data, err := os.ReadFile("log/enc/app.log")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "abc") {
		t.Fatalf("token should be masked in file: %s", data)
	}
}

func TestLogItemJSONRoundTrip(t *testing.T) {
	item := &LogItem{
		Level:      InfoLevel,
		RequestID:  "r",
		Message:    "m",
		CreateTime: "2020-01-01 00:00:00.000000",
	}
	buf := bytes.NewBuffer(nil)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(item); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"message":"m"`) {
		t.Fatal(buf.String())
	}
}
