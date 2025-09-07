package cmd

import (
	"strings"
	"testing"
)

// Ensure renderers preserve input order (no re-sorting), using IDs as markers.
func TestRenderItems_PreservesOrder(t *testing.T) {
	items := []queryItem{
		{ID: 200, Fields: map[string]any{"System.Title": "B", "System.State": "New", "System.WorkItemType": "Task"}},
		{ID: 100, Fields: map[string]any{"System.Title": "A", "System.State": "New", "System.WorkItemType": "Task"}},
	}
	md, err := renderItems(items)
	if err != nil {
		t.Fatalf("renderItems error: %v", err)
	}
	first := strings.Index(md, "| 200 |")
	second := strings.Index(md, "| 100 |")
	if first < 0 || second < 0 || !(first < second) {
		t.Fatalf("order not preserved in markdown: %q", md)
	}
}

func TestRenderItemsTypeLess_PreservesOrder(t *testing.T) {
	items := []queryItem{
		{ID: 3, Fields: map[string]any{"System.Title": "X", "System.State": "Active"}},
		{ID: 1, Fields: map[string]any{"System.Title": "Y", "System.State": "Active"}},
	}
	md, err := renderItemsTypeLess(items, "")
	if err != nil {
		t.Fatalf("renderItemsTypeLess error: %v", err)
	}
	first := strings.Index(md, "| 3 |")
	second := strings.Index(md, "| 1 |")
	if first < 0 || second < 0 || !(first < second) {
		t.Fatalf("order not preserved in markdown: %q", md)
	}
}
