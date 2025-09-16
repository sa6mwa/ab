//go:build windows
// +build windows

package openurl

import (
	"os/exec"
)

// Open opens the given URL on Windows using 'rundll32 url.dll,FileProtocolHandler'.
func Open(url string) error {
	// Use start via cmd /c start would require shell escaping; rundll32 is simpler.
	return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}
