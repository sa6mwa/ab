package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"

	azpkg "github.com/sa6mwa/ab/internal/az"
)

// fake work item JSON builder
func wiJSON(id string, col string) []byte {
	// coerce id to int if possible
	var idNum any = id
	if n, err := strconv.Atoi(id); err == nil {
		idNum = n
	}
	m := map[string]any{
		"id": idNum,
		"fields": map[string]any{
			"System.State":          "New",
			"System.Title":          "Title",
			"System.WorkItemType":   "User Story",
			"WEF_ABC_Kanban.Column": col,
			"System.Tags":           "",
		},
		"url": fmt.Sprintf("https://example/_apis/wit/workItems/%s", id),
	}
	b, _ := json.Marshal(m)
	return b
}

func TestForward_MovesToNextColumn(t *testing.T) {
	_ = azpkg.SetConfirmMode("never")
	defer azpkg.SetExecutorForTest(nil)
	called := 0
	azpkg.SetExecutorForTest(func(args ...string) ([]byte, error) {
		called++
		if len(args) >= 3 && args[0] == "boards" && args[1] == "work-item" && args[2] == "show" {
			return wiJSON(args[indexOf(args, "--id")+1], "Backlog"), nil
		}
		if len(args) >= 3 && args[0] == "boards" && args[1] == "work-item" && args[2] == "update" {
			// Ensure fields contain our kanban column with next value
			fi := indexOf(args, "--fields")
			if fi < 0 || fi+1 >= len(args) {
				t.Fatalf("--fields not found in args: %v", args)
			}
			fields := args[fi+1 : len(args)]
			// until -o
			if oi := indexOf(args, "-o"); oi > fi {
				fields = args[fi+1 : oi]
			}
			found := false
			for _, f := range fields {
				if strings.HasPrefix(f, "WEF_ABC_Kanban.Column=") {
					if !strings.HasSuffix(f, "Ready for Development") {
						t.Fatalf("expected next column Ready for Development, got %s", f)
					}
					found = true
				}
			}
			if !found {
				t.Fatalf("kanban column field update not found in %v", fields)
			}
			return wiJSON(args[indexOf(args, "--id")+1], "Ready for Development"), nil
		}
		t.Fatalf("unexpected az exec args: %v", args)
		return nil, nil
	})
	if err := forwardCmd.RunE(forwardCmd, []string{"123"}); err != nil {
		t.Fatalf("forward error: %v", err)
	}
	if called < 2 {
		t.Fatalf("expected at least 2 az calls, got %d", called)
	}
}

func TestBackward_MovesToPrevColumn(t *testing.T) {
	_ = azpkg.SetConfirmMode("never")
	defer azpkg.SetExecutorForTest(nil)
	azpkg.SetExecutorForTest(func(args ...string) ([]byte, error) {
		if len(args) >= 3 && args[0] == "boards" && args[1] == "work-item" && args[2] == "show" {
			return wiJSON(args[indexOf(args, "--id")+1], "Ready for Development"), nil
		}
		if len(args) >= 3 && args[0] == "boards" && args[1] == "work-item" && args[2] == "update" {
			fi := indexOf(args, "--fields")
			fields := args[fi+1:]
			if oi := indexOf(args, "-o"); oi > fi {
				fields = args[fi+1 : oi]
			}
			ok := false
			for _, f := range fields {
				if f == "WEF_ABC_Kanban.Column=Backlog" {
					ok = true
				}
			}
			if !ok {
				t.Fatalf("expected update to Backlog, got fields %v", fields)
			}
			return wiJSON(args[indexOf(args, "--id")+1], "Backlog"), nil
		}
		t.Fatalf("unexpected az exec args: %v", args)
		return nil, nil
	})
	if err := backwardCmd.RunE(backwardCmd, []string{"456"}); err != nil {
		t.Fatalf("backward error: %v", err)
	}
}

func TestForward_AtDoneErrors(t *testing.T) {
	_ = azpkg.SetConfirmMode("never")
	defer azpkg.SetExecutorForTest(nil)
	azpkg.SetExecutorForTest(func(args ...string) ([]byte, error) {
		if len(args) >= 3 && args[0] == "boards" && args[1] == "work-item" && args[2] == "show" {
			return wiJSON(args[indexOf(args, "--id")+1], "Done"), nil
		}
		t.Fatalf("update should not be called when at Done: %v", args)
		return nil, nil
	})
	if err := forwardCmd.RunE(forwardCmd, []string{"999"}); err == nil {
		t.Fatalf("expected error when forwarding from Done")
	}
}

func TestBackward_AtBacklogErrors(t *testing.T) {
	_ = azpkg.SetConfirmMode("never")
	defer azpkg.SetExecutorForTest(nil)
	azpkg.SetExecutorForTest(func(args ...string) ([]byte, error) {
		if len(args) >= 3 && args[0] == "boards" && args[1] == "work-item" && args[2] == "show" {
			return wiJSON(args[indexOf(args, "--id")+1], "Backlog"), nil
		}
		t.Fatalf("update should not be called when at Backlog")
		return nil, nil
	})
	if err := backwardCmd.RunE(backwardCmd, []string{"1000"}); err == nil {
		t.Fatalf("expected error when moving back from Backlog")
	}
}

func indexOf(s []string, v string) int {
	for i, x := range s {
		if x == v {
			return i
		}
	}
	return -1
}
