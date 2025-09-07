package term

import (
	"os"
	"testing"
)

// save/restore helpers for function variables
func withFuncs(t *testing.T, native func() int, stty func() int, cols func() int) func() {
	oldNative := nativeWidthFn
	oldStty := sttyWidthFn
	oldCols := columnsEnvFn
	nativeWidthFn = native
	sttyWidthFn = stty
	columnsEnvFn = cols
	return func() {
		nativeWidthFn = oldNative
		sttyWidthFn = oldStty
		columnsEnvFn = oldCols
	}
}

func TestDetectWidth_PrefersNative(t *testing.T) {
	restore := withFuncs(t, func() int { return 123 }, func() int { return 77 }, func() int { return 66 })
	defer restore()
	if got := DetectWidth(); got != 123 {
		t.Fatalf("want 123 from native, got %d", got)
	}
}

func TestDetectWidth_FallbackStty(t *testing.T) {
	restore := withFuncs(t, func() int { return 0 }, func() int { return 88 }, func() int { return 66 })
	defer restore()
	if got := DetectWidth(); got != 88 {
		t.Fatalf("want 88 from stty, got %d", got)
	}
}

func TestDetectWidth_FallbackColumns(t *testing.T) {
	restore := withFuncs(t, func() int { return 0 }, func() int { return 0 }, func() int { return 99 })
	defer restore()
	if got := DetectWidth(); got != 99 {
		t.Fatalf("want 99 from COLUMNS, got %d", got)
	}
}

func TestDetectWidth_Default80(t *testing.T) {
	restore := withFuncs(t, func() int { return 0 }, func() int { return 0 }, func() int { return 0 })
	defer restore()
	if got := DetectWidth(); got != 80 {
		t.Fatalf("want default 80, got %d", got)
	}
}

func TestColumnsEnv_ReadsEnv(t *testing.T) {
	// This tests our columnsEnv() helper without relying on native/stty.
	t.Setenv("COLUMNS", "120")
	if got := columnsEnv(); got != 120 {
		t.Fatalf("want 120 from env, got %d", got)
	}
	// Invalid values should return 0, leading to default in DetectWidth order
	t.Setenv("COLUMNS", "not-a-number")
	if got := columnsEnv(); got != 0 {
		t.Fatalf("want 0 for invalid env, got %d", got)
	}
	os.Unsetenv("COLUMNS")
	if got := columnsEnv(); got != 0 {
		t.Fatalf("want 0 for unset env, got %d", got)
	}
}
