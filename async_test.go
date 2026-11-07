package logging

import (
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAsyncCore_WriteAndClose(t *testing.T) {
	WrittenLogs.Store(0)
	sink := &encodeSink{output: OutputStd, writer: discardWriter{}}
	a := newAsyncCore(sink, 128, true)
	if err := a.Write(&LogItem{Message: "async-one"}, false); err != nil {
		t.Fatal(err)
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	if WrittenLogs.Load() < 1 {
		t.Fatalf("WrittenLogs = %d", WrittenLogs.Load())
	}
}

func TestAsyncCore_DropOnFull(t *testing.T) {
	DroppedLogs.Store(0)
	WrittenLogs.Store(0)

	// Block the sink so the worker stalls and the queue can fill and drop.
	gate := make(chan struct{})
	sink := &encodeSink{output: OutputStd, writer: &blockingWriter{gate: gate}}
	a := newAsyncCore(sink, 2, true)
	if err := a.Write(&LogItem{Message: "block"}, false); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(200 * time.Millisecond)
	for WrittenLogs.Load() < 1 && time.Now().Before(deadline) {
		time.Sleep(2 * time.Millisecond)
	}
	for i := 0; i < 20; i++ {
		_ = a.Write(&LogItem{Message: "x"}, false)
	}
	close(gate)
	_ = a.Close()
	if DroppedLogs.Load() == 0 {
		t.Fatal("expected some drops")
	}
}

type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }

// blockingWriter blocks the first Write until gate closes (queue drop tests).
type blockingWriter struct {
	gate <-chan struct{}
}

func (w *blockingWriter) Write(p []byte) (int, error) {
	<-w.gate
	return len(p), nil
}

func TestMaskLogItem_beforeEncode(t *testing.T) {
	if err := SetupFromConfig(Config{
		Output:      OutputStd,
		AESDisabled: false,
		AESFields:   []string{"password"},
		AsyncSync:   true,
	}); err != nil {
		t.Fatal(err)
	}
	defer Close()

	out := captureStdout(func() {
		New().Info("mask", map[string]string{"password": "secret123"}, nil)
	})
	if strings.Contains(out, "secret123") {
		t.Fatalf("password should be masked: %s", out)
	}
}

func TestAsyncFmtOutputStaysSync(t *testing.T) {
	if err := SetupFromConfig(Config{Output: OutputFmt, AESDisabled: true}); err != nil {
		t.Fatal(err)
	}
	defer Close()
	_, ok := defaultCore.(*encodeSink)
	if !ok {
		t.Fatalf("fmt should use sync encodeSink, got %T", defaultCore)
	}
}

// Integration: allow the worker time to drain the queue.
func TestAsyncIntegration(t *testing.T) {
	WrittenLogs.Store(0)
	if err := SetupFromConfig(Config{
		Output:         OutputStd,
		AESDisabled:    true,
		AsyncQueueSize: 64,
	}); err != nil {
		t.Fatal(err)
	}
	writeLog(&LogItem{Message: "async-int"}, false)
	deadline := time.Now().Add(500 * time.Millisecond)
	for WrittenLogs.Load() < 1 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	_ = Close()
	if WrittenLogs.Load() < 1 {
		t.Fatal("expected written log")
	}
}

func TestAsyncCore_WriteAfterCloseNoPanic(t *testing.T) {
	sink := &encodeSink{output: OutputStd, writer: discardWriter{}}
	a := newAsyncCore(sink, 8, true)
	if err := a.Write(&LogItem{Message: "before"}, false); err != nil {
		t.Fatal(err)
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 100; i++ {
		if err := a.Write(&LogItem{Message: "after"}, false); err != nil {
			t.Fatal(err)
		}
	}
}

func TestClose_ConcurrentWriteNoPanic(t *testing.T) {
	sink := &encodeSink{output: OutputStd, writer: discardWriter{}}
	a := newAsyncCore(sink, 64, true)
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_ = a.Write(&LogItem{Message: "race"}, false)
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = a.Close()
	}()
	wg.Wait()
}

func TestSetupFromConfig_twiceNoGoroutineLeak(t *testing.T) {
	baseline := runtime.NumGoroutine()
	for i := 0; i < 3; i++ {
		if err := SetupFromConfig(Config{
			Output:         OutputStd,
			AESDisabled:    true,
			AsyncQueueSize: 16,
		}); err != nil {
			t.Fatal(err)
		}
		if err := Close(); err != nil {
			t.Fatal(err)
		}
	}
	deadline := time.Now().Add(500 * time.Millisecond)
	for runtime.NumGoroutine() > baseline+2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if n := runtime.NumGoroutine(); n > baseline+2 {
		t.Fatalf("goroutines after 3x setup/close: got %d baseline %d", n, baseline)
	}
}
