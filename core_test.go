package logging

import (
	"sync"
	"testing"
)

func TestClose_idempotent(t *testing.T) {
	if err := SetupFromConfig(Config{Output: OutputStd, AESDisabled: true, AsyncSync: true}); err != nil {
		t.Fatal(err)
	}
	if err := Close(); err != nil {
		t.Fatal(err)
	}
	if err := Close(); err != nil {
		t.Fatal(err)
	}
	if IsSetup {
		t.Fatal("IsSetup should be false after Close")
	}
}

func TestSetupClose_concurrentWrite(t *testing.T) {
	if err := SetupFromConfig(Config{
		Output:         OutputStd,
		AESDisabled:    true,
		AsyncQueueSize: 128,
	}); err != nil {
		t.Fatal(err)
	}
	defer Close()

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				writeLog(&LogItem{Message: "concurrent"}, false)
			}
		}()
	}
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = Close()
		}()
	}
	wg.Wait()
}

func TestLevelConstants_matchJSON(t *testing.T) {
	if LevelFatal != int(FatalLevel) || LevelInfo != int(InfoLevel) {
		t.Fatalf("level constants mismatch: fatal=%d info=%d", LevelFatal, LevelInfo)
	}
}
