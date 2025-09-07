package board

import "testing"

func TestNextPrevColumn(t *testing.T) {
	// Use a simple order
	ColumnOrder = []string{"Backlog", "Active", "Done"}

	n, err := NextColumn("Backlog")
	if err != nil || n != "Active" {
		t.Fatalf("NextColumn Backlog => %q, %v", n, err)
	}
	if _, err := NextColumn("Done"); err == nil {
		t.Fatalf("expected error on last column")
	}
	p, err := PrevColumn("Done")
	if err != nil || p != "Active" {
		t.Fatalf("PrevColumn Done => %q, %v", p, err)
	}
	if _, err := PrevColumn("Backlog"); err == nil {
		t.Fatalf("expected error on first column")
	}
	if _, err := NextColumn("Unknown"); err == nil {
		t.Fatalf("expected error on unknown column")
	}
}

func TestSetColumnOrderHelpers(t *testing.T) {
	// CSV parsing
	if err := SetColumnOrderFromCSV("A, B ,C"); err != nil {
		t.Fatalf("SetColumnOrderFromCSV error: %v", err)
	}
	if len(ColumnOrder) != 3 || ColumnOrder[1] != "B" {
		t.Fatalf("ColumnOrder not set correctly: %#v", ColumnOrder)
	}
	// Default Agile
	SetDefaultAgileColumns()
	if len(ColumnOrder) != 4 || ColumnOrder[0] != "New" || ColumnOrder[3] != "Closed" {
		t.Fatalf("SetDefaultAgileColumns unexpected: %#v", ColumnOrder)
	}
}
