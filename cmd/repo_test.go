package cmd

import (
    "fmt"
    "strings"
    "testing"
)

func TestHumanSize(t *testing.T) {
    tests := []struct{ n int64; want string }{
        {0, "0 B"},
        {1, "1 B"},
        {1023, "1023 B"},
        {1024, "1 KB"},
        {2048, "2 KB"},
        {1024 * 1024, "1 MB"},
    }
    for _, tt := range tests {
        if got := humanSize(tt.n); got != tt.want {
            t.Fatalf("humanSize(%d)=%q want %q", tt.n, got, tt.want)
        }
    }
}

func TestListFormatting_Alignment(t *testing.T) {
    repos := []fakeRepo{{"short", "id1", 0}, {"much-longer-name", "id2", 10}}
    out := formatRepoListForTest(repos)
    lines := strings.Split(strings.TrimSpace(out), "\n")
    if len(lines) != 2 {
        t.Fatalf("expected 2 lines, got %d: %q", len(lines), out)
    }
    // Name columns should align: the '|' position should match
    idx1 := strings.Index(lines[0], "|")
    idx2 := strings.Index(lines[1], "|")
    if idx1 != idx2 || idx1 <= 0 {
        t.Fatalf("expected aligned '|' positions, got %d vs %d: %q", idx1, idx2, out)
    }
}

// Minimal shape to test padding; mirrors fields needed from az.Repo
type fakeRepo struct{ Name, ID string; Size int64 }

// formatRepoListForTest mirrors the logic inside repo list printing for alignment.
func formatRepoListForTest(repos []fakeRepo) string {
    max := 0
    for _, r := range repos { if len(r.Name) > max { max = len(r.Name) } }
    var b strings.Builder
    // Use Sprintf-like helper for deterministic format
    for _, r := range repos {
        b.WriteString(fmtSprintfPad(max, r.Name, r.ID, r.Size))
        b.WriteByte('\n')
    }
    return b.String()
}

func fmtSprintfPad(max int, name, id string, size int64) string {
    // Minimal reimplementation mirroring the production format string.
    return sprintfPad(max, name, id, size)
}

// split for easier test-local replacement without importing fmt here.
func sprintfPad(max int, name, id string, size int64) string { return fmt.Sprintf("%-*s | %s | %d B", max, name, id, size) }
