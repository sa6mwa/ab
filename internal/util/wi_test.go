package util

import "testing"

func TestFindKanbanColumn(t *testing.T) {
	fields := map[string]interface{}{
		"System.Title":             "x",
		"WEF_ABC123_Kanban.Column": "In Process",
		"Microsoft.VSTS.State":     "Active",
	}
	name, val := FindKanbanColumn(fields)
	if name == "" || val != "In Process" {
		t.Fatalf("FindKanbanColumn got %q=%q", name, val)
	}
}

func TestFieldString(t *testing.T) {
	fields := map[string]interface{}{"a": "b", "n": 1}
	if FieldString(fields, "a") != "b" {
		t.Fatalf("FieldString string failed")
	}
	if FieldString(fields, "n") != "" {
		t.Fatalf("FieldString non-string should be empty")
	}
	if FieldString(nil, "x") != "" {
		t.Fatalf("FieldString nil should be empty")
	}
}
