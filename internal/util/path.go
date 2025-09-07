package util

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// ExpandTilde expands leading ~ and ~user in a file path.
// - "~" or "~/..." expands to the current user's home directory.
// - "~user" or "~user/..." expands to the named user's home directory.
// If the input doesn't start with '~' or is empty, it is returned unchanged.
func ExpandTilde(p string) (string, error) {
	if strings.TrimSpace(p) == "" || p[0] != '~' {
		return p, nil
	}
	// Handle current user variants: ~ or ~/...
	if p == "~" {
		h, err := os.UserHomeDir()
		if err != nil {
			return p, err
		}
		return h, nil
	}
	if strings.HasPrefix(p, "~/") {
		h, err := os.UserHomeDir()
		if err != nil {
			return p, err
		}
		return filepath.Join(h, p[2:]), nil
	}
	// Handle ~username or ~username/...
	rest := p[1:]
	slash := strings.IndexByte(rest, '/')
	var uname, tail string
	if slash < 0 {
		uname = rest
		tail = ""
	} else {
		uname = rest[:slash]
		tail = rest[slash+1:]
	}
	u, err := user.Lookup(uname)
	if err != nil {
		return p, err
	}
	if tail == "" {
		return u.HomeDir, nil
	}
	return filepath.Join(u.HomeDir, tail), nil
}
