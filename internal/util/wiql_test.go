package util

import "testing"

func TestParseWIQLIDs_Shape1_ObjectWorkItems(t *testing.T) {
	raw := []byte(`{"workItems":[{"id":1},{"id":2}]}`)
	ids, err := ParseWIQLIDs(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 2 || ids[0] != 1 || ids[1] != 2 {
		t.Fatalf("got %v", ids)
	}
}

func TestParseWIQLIDs_Shape2_Array(t *testing.T) {
	raw := []byte(`[{"id":3},{"id":4}]`)
	ids, err := ParseWIQLIDs(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 2 || ids[0] != 3 || ids[1] != 4 {
		t.Fatalf("got %v", ids)
	}
}

func TestParseWIQLIDs_Shape3_Value(t *testing.T) {
	raw := []byte(`{"value":[{"id":5},{"id":6}]}`)
	ids, err := ParseWIQLIDs(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 2 || ids[0] != 5 || ids[1] != 6 {
		t.Fatalf("got %v", ids)
	}
}

func TestParseWIQLIDs_Unknown(t *testing.T) {
	raw := []byte(`{"foo":"bar"}`)
	if _, err := ParseWIQLIDs(raw); err == nil {
		t.Fatalf("expected error for unknown shape")
	}
}
