package util

import (
	"os"
	osuser "os/user"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpandTilde_CurrentUser(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir available")
	}
	got, err := ExpandTilde("~")
	if err != nil {
		t.Fatalf("ExpandTilde(~) error: %v", err)
	}
	if got != home {
		t.Fatalf("ExpandTilde(~) = %q, want %q", got, home)
	}

	got2, err := ExpandTilde("~/foo/bar")
	if err != nil {
		t.Fatalf("ExpandTilde(~/foo/bar) error: %v", err)
	}
	want2 := filepath.Join(home, "foo", "bar")
	if got2 != want2 {
		t.Fatalf("ExpandTilde(~/foo/bar) = %q, want %q", got2, want2)
	}
}

func TestExpandTilde_NamedUser(t *testing.T) {
	u, err := osuser.Current()
	if err != nil || strings.TrimSpace(u.Username) == "" {
		t.Skip("unable to get current user")
	}
	got, err := ExpandTilde("~" + u.Username + "/docs")
	if err != nil {
		t.Fatalf("ExpandTilde(~user/docs) error: %v", err)
	}
	want := filepath.Join(u.HomeDir, "docs")
	if got != want {
		t.Fatalf("ExpandTilde(~user/docs) = %q, want %q", got, want)
	}
}

func TestExpandTilde_NoChange(t *testing.T) {
	in := "/tmp/file.txt"
	got, err := ExpandTilde(in)
	if err != nil {
		t.Fatalf("ExpandTilde returned error: %v", err)
	}
	if got != in {
		t.Fatalf("ExpandTilde changed non-tilde path: %q -> %q", in, got)
	}
}
