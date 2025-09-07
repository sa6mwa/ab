//go:build unix && !js

package term

import (
	"os"
	"os/exec"
	"strconv"
	"strings"

	xterm "golang.org/x/term"
)

// widthNative tries to detect terminal width using golang.org/x/term on Unix.
func widthNative() int {
	fd := int(os.Stdout.Fd())
	w, _, err := xterm.GetSize(fd)
	if err == nil && w > 0 {
		return w
	}
	return 0
}

// sttyWidth uses `stty size` as a fallback on Unix.
func sttyWidth() int {
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	parts := strings.Fields(string(out))
	if len(parts) == 2 {
		if n, err := strconv.Atoi(parts[1]); err == nil && n > 0 {
			return n
		}
	}
	return 0
}
