package cmd

import (
	"strings"
	"testing"

	azpkg "github.com/sa6mwa/ab/internal/az"
)

// Ensure PO order handles empty result sets without error.
func TestQueryPOOrdered_EmptyResults(t *testing.T) {
	_ = azpkg.SetConfirmMode("never")
	defer azpkg.SetExecutorForTest(nil)
	// Stub az executor to always return an empty array JSON
	azpkg.SetExecutorForTest(func(args ...string) ([]byte, error) {
		return []byte("[]"), nil
	})
	items, err := queryPOOrdered(false)
	if err != nil {
		t.Fatalf("queryPOOrdered error: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
}

func TestQueryPOOrdered_EmptyObjectResults(t *testing.T) {
	_ = azpkg.SetConfirmMode("never")
	defer azpkg.SetExecutorForTest(nil)
	// Return an object shape with empty arrays
	azpkg.SetExecutorForTest(func(args ...string) ([]byte, error) {
		return []byte(`{"workItems":[]}`), nil
	})
	items, err := queryPOOrdered(true)
	if err != nil {
		t.Fatalf("queryPOOrdered error: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
}

// Mixed-case: stories/bugs returns items and others empty
func TestQueryPOOrdered_Mixed_FirstHasItems(t *testing.T) {
	_ = azpkg.SetConfirmMode("never")
	defer azpkg.SetExecutorForTest(nil)
	azpkg.SetExecutorForTest(func(args ...string) ([]byte, error) {
		// args include: boards query --wiql <WIQL> -o json
		for i := 0; i < len(args); i++ {
			if args[i] == "--wiql" && i+1 < len(args) {
				wiql := args[i+1]
				if strings.Contains(wiql, "NOT IN ('User Story','Bug')") {
					return []byte(`[]`), nil
				}
				if strings.Contains(wiql, " IN ('User Story','Bug')") {
					// return one item in array shape
					return []byte(`[{"id": 101, "fields": {"System.Title":"A","System.WorkItemType":"Bug","System.State":"New"}}]`), nil
				}
			}
		}
		return []byte("[]"), nil
	})
	items, err := queryPOOrdered(false)
	if err != nil {
		t.Fatalf("queryPOOrdered error: %v", err)
	}
	if len(items) != 1 || items[0].ID != 101 {
		t.Fatalf("expected 1 item id=101, got %#v", items)
	}
}

// Mixed-case: stories/bugs empty and others returns items
func TestQueryPOOrdered_Mixed_SecondHasItems(t *testing.T) {
	_ = azpkg.SetConfirmMode("never")
	defer azpkg.SetExecutorForTest(nil)
	azpkg.SetExecutorForTest(func(args ...string) ([]byte, error) {
		for i := 0; i < len(args); i++ {
			if args[i] == "--wiql" && i+1 < len(args) {
				wiql := args[i+1]
				if strings.Contains(wiql, "NOT IN ('User Story','Bug')") {
					// return two items in object shape under workItems
					return []byte(`{"workItems":[
                        {"id": 202, "fields": {"System.Title":"X","System.WorkItemType":"Task","System.State":"Active"}},
                        {"id": 201, "fields": {"System.Title":"Y","System.WorkItemType":"Task","System.State":"Active"}}
                    ]}`), nil
				}
				if strings.Contains(wiql, " IN ('User Story','Bug')") {
					return []byte(`{"workItems":[]}`), nil
				}
			}
		}
		return []byte(`[]`), nil
	})
	items, err := queryPOOrdered(true)
	if err != nil {
		t.Fatalf("queryPOOrdered error: %v", err)
	}
	if len(items) != 2 || items[0].ID != 202 || items[1].ID != 201 {
		t.Fatalf("expected 2 items 202,201 got %#v", items)
	}
}
