package az

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// helper to stub executor
func withStubExec(t *testing.T, f func(args ...string) ([]byte, error), test func()) {
	t.Helper()
	prev := azExec
	azExec = f
	defer func() { azExec = prev }()
	test()
}

func TestFormatAz_Quoting(t *testing.T) {
    got := formatAz([]string{"boards", "query", "--wiql", "SELECT 'a b'", "-o", "json"})
    // Using shellescape now; expected pattern to embed single quotes is: '"'"'
    if !strings.Contains(got, `'SELECT '"'"'a b'"'"''`) {
        t.Fatalf("expected shell-escaped wiql, got: %s", got)
    }
}

func TestConfirmModes_ShouldConfirm(t *testing.T) {
	// Always
	_ = SetConfirmMode("always")
	if !shouldConfirm([]string{"boards", "query"}) {
		t.Fatal("always should confirm")
	}
	// Never
	_ = SetConfirmMode("never")
	if shouldConfirm([]string{"boards", "query"}) {
		t.Fatal("never should not confirm")
	}
	// Mutations
	_ = SetConfirmMode("mutations")
	if !shouldConfirm([]string{"boards", "work-item", "create"}) {
		t.Fatal("create should confirm in mutations mode")
	}
	if !shouldConfirm([]string{"boards", "work-item", "update"}) {
		t.Fatal("update should confirm in mutations mode")
	}
	if shouldConfirm([]string{"boards", "work-item", "show"}) {
		t.Fatal("show should not confirm in mutations mode")
	}
	if shouldConfirm([]string{"boards", "query"}) {
		t.Fatal("query should not confirm in mutations mode")
	}
	if shouldConfirm([]string{"rest", "--method", "GET", "--url", "http://example"}) {
		t.Fatal("rest GET should not confirm in mutations mode")
	}
	if !shouldConfirm([]string{"rest", "--method", "POST", "--url", "http://example"}) {
		t.Fatal("rest POST should confirm in mutations mode")
	}
}

func TestRun_QueryUsesExecutor_NoPromptWhenNever(t *testing.T) {
	_ = SetConfirmMode("never")
	calls := 0
	var gotArgs []string
	withStubExec(t, func(args ...string) ([]byte, error) {
		calls++
		gotArgs = append([]string(nil), args...)
		// Return minimal JSON that callers can parse
		return json.RawMessage(`{"workItems":[]}`), nil
	}, func() {
		_, err := QueryWIQL("SELECT 1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if calls != 1 {
			t.Fatalf("expected 1 call, got %d", calls)
		}
		if len(gotArgs) == 0 || gotArgs[0] != "boards" {
			t.Fatalf("unexpected args: %v", gotArgs)
		}
	})
}

func TestUpdateWorkItemFields_BuildsArgs(t *testing.T) {
	_ = SetConfirmMode("never")
	var captured []string
	withStubExec(t, func(args ...string) ([]byte, error) {
		captured = append([]string(nil), args...)
		return json.RawMessage(`{}`), nil
	}, func() {
		_, err := UpdateWorkItemFields("123", map[string]string{
			"System.State":      "Active",
			"System.AssignedTo": "me@example.com",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !containsAll(captured, []string{"boards", "work-item", "update", "--id", "123", "--fields"}) {
			t.Fatalf("args missing expected tokens: %v", captured)
		}
		// Ensure both field assignments present (order not guaranteed)
		if !(contains(captured, "System.State=Active") && contains(captured, "System.AssignedTo=me@example.com")) {
			t.Fatalf("missing field args: %v", captured)
		}
	})
}

func TestCreateWorkItem_BuildsArgs(t *testing.T) {
	_ = SetConfirmMode("never")
	var captured []string
	withStubExec(t, func(args ...string) ([]byte, error) {
		captured = append([]string(nil), args...)
		return json.RawMessage(`{}`), nil
	}, func() {
		_, err := CreateWorkItem("Task", "Title", map[string]string{"System.State": "New"}, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !containsAll(captured, []string{"boards", "work-item", "create", "--type", "Task", "--title", "Title", "--fields", "System.State=New", "-o", "json"}) {
			t.Fatalf("args missing tokens: %v", captured)
		}
	})
}

func TestAddWorkItemRelation_BuildsArgs(t *testing.T) {
	_ = SetConfirmMode("never")
	var captured []string
	withStubExec(t, func(args ...string) ([]byte, error) {
		captured = append([]string(nil), args...)
		return json.RawMessage(`{}`), nil
	}, func() {
		_, err := AddWorkItemRelation("123", "parent", "42")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !containsAll(captured, []string{"boards", "work-item", "relation", "add", "--id", "123", "--relation-type", "parent", "--target-id", "42", "-o", "json"}) {
			t.Fatalf("args missing tokens: %v", captured)
		}
	})
}

func TestSetConfirmMode_Parse(t *testing.T) {
	tests := map[string]ConfirmMode{
		"always":    ConfirmAlways,
		"mutations": ConfirmMutations,
		"never":     ConfirmNever,
		"on":        ConfirmAlways,
		"off":       ConfirmNever,
		"true":      ConfirmAlways,
		"false":     ConfirmNever,
	}
	for s, want := range tests {
		if err := SetConfirmMode(s); err != nil {
			t.Fatalf("unexpected error for %s: %v", s, err)
		}
		if confirmMode != want {
			t.Fatalf("mode for %s = %v, want %v", s, confirmMode, want)
		}
	}
	if err := SetConfirmMode("wat"); err == nil {
		t.Fatal("expected error for invalid mode")
	}
}

func containsAll(s, subs []string) bool {
	for _, sub := range subs {
		if !contains(s, sub) {
			return false
		}
	}
	return true
}
func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// sanity to ensure PrintJSON doesn't error on newline-less data
func TestPrintJSON_Newline(t *testing.T) {
	// Just verify it doesn't panic or error when writing
	if err := PrintJSON([]byte("{}")); err != nil && !errors.Is(err, nil) {
		t.Fatalf("unexpected error: %v", err)
	}
}
