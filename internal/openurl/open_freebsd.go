//go:build freebsd
// +build freebsd

package openurl

import "os/exec"

// Open opens the given URL on FreeBSD using xdg-open if available,
// otherwise tries 'open' if present.
func Open(url string) error {
    if _, err := exec.LookPath("xdg-open"); err == nil {
        return exec.Command("xdg-open", url).Start()
    }
    if _, err := exec.LookPath("open"); err == nil {
        return exec.Command("open", url).Start()
    }
    return exec.Command("xdg-open", url).Start()
}

