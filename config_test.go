package logging

import (
	"errors"
	"testing"
)

func TestSetupFromConfig(t *testing.T) {
	cfg := Config{
		Output:      OutputStd,
		AESDisabled: true,
		AsyncSync:   true,
	}
	err := SetupFromConfig(cfg)
	if err != nil {
		t.Fatalf("SetupFromConfig err = %v", err)
	}
	t.Cleanup(func() { _ = Close() })
	if !IsSetup {
		t.Error("IsSetup should be true")
	}
}

func TestSetupFromConfig_RejectsAESKey(t *testing.T) {
	cfg := Config{
		Output:    OutputStd,
		AESKey:    "0123456789abcdef",
		AESFields: []string{"secret"},
	}
	err := SetupFromConfig(cfg)
	if !errors.Is(err, ErrAESKeyUnsupported) {
		t.Fatalf("err = %v, want ErrAESKeyUnsupported", err)
	}
}

func TestSetupFromConfig_InvalidAESField(t *testing.T) {
	cfg := Config{
		Output:    OutputStd,
		AESFields: []string{"invalid-field!"},
	}
	err := SetupFromConfig(cfg)
	if err == nil {
		t.Error("expected error for invalid mask field")
	}
}
