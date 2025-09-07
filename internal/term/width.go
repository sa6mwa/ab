package term

import (
	"os"
	"strconv"
)

// DetectWidth returns the terminal width using a native per-OS method,
// then falls back to stty (on Unix), then $COLUMNS, and finally 80.
func DetectWidth() int {
	if w := nativeWidthFn(); w > 0 {
		return w
	}
	if w := sttyWidthFn(); w > 0 { // no-op on non-Unix
		return w
	}
	if n := columnsEnvFn(); n > 0 {
		return n
	}
	return 80
}

// widthNative is implemented per-OS (see width_unix.go and width_windows.go).
// sttyWidth is implemented on Unix; on non-Unix it returns 0.

// function indirections to ease testing precedence without relying on OS/io.
var (
	nativeWidthFn = widthNative
	sttyWidthFn   = sttyWidth
	columnsEnvFn  = columnsEnv
)

func columnsEnv() int {
	if c := os.Getenv("COLUMNS"); c != "" {
		if n, err := strconv.Atoi(c); err == nil && n > 0 {
			return n
		}
	}
	return 0
}
