//go:build windows

package term

import (
	"os"

	xterm "golang.org/x/term"
)

// widthNative uses golang.org/x/term on Windows.
func widthNative() int {
	fd := int(os.Stdout.Fd())
	w, _, err := xterm.GetSize(fd)
	if err == nil && w > 0 {
		return w
	}
	return 0
}

// sttyWidth is not applicable on Windows.
func sttyWidth() int { return 0 }
