//go:build linux
// +build linux

package openurl

import (
	"os/exec"
)

// Open opens the given URL using the desktop's opener.
func Open(url string) error {
	// Try xdg-open which is common on Linux; fall back to 'open' if present.
	if _, err := exec.LookPath("xdg-open"); err == nil {
		return exec.Command("xdg-open", url).Start()
	}
	if _, err := exec.LookPath("open"); err == nil {
		return exec.Command("open", url).Start()
	}
	return exec.Command("/usr/bin/xdg-open", url).Start()
}
