package git

import (
	shellescape "al.essio.dev/pkg/shellescape"
	"fmt"
	"os"
	"os/exec"
)

// Clone runs `git clone <url>` wiring stdio. Prints the command unless silent.
func Clone(url string, silent bool) error {
	if url == "" {
		return fmt.Errorf("empty clone URL")
	}
	if !silent {
		fmt.Fprintln(os.Stderr, shellescape.QuoteCommand([]string{"git", "clone", url}))
	}
	cmd := exec.Command("git", "clone", url)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// keep file-local funcs minimal; shellescape handles printing, not execution
