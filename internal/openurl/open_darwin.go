//go:build darwin
// +build darwin

package openurl

import "os/exec"

// Open opens the given URL on macOS using the 'open' command.
func Open(url string) error {
	return exec.Command("open", url).Start()
}
