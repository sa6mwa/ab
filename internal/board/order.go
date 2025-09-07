package board

import (
	"fmt"
	"os"
	"strings"
)

// ColumnOrder defines the canonical Kanban column progression used by this CLI.
var ColumnOrder = []string{
	"Backlog",
	"Ready for Development",
	"In Process",
	"Ready to Test",
	"In Test",
	"Deploy",
	"Done",
}

// init loads column order from AB_COLUMNS if provided.
func init() {
	if csv := strings.TrimSpace(os.Getenv("AB_COLUMNS")); csv != "" {
		_ = SetColumnOrderFromCSV(csv)
	}
}

// SetDefaultAgileColumns sets the column order to the default Agile process columns.
func SetDefaultAgileColumns() {
	ColumnOrder = []string{"New", "Active", "Resolved", "Closed"}
}

// SetColumnOrderFromCSV parses a comma-separated string and sets ColumnOrder accordingly.
// Empty entries are ignored; surrounding whitespace is trimmed. Returns an error if no valid columns were found.
func SetColumnOrderFromCSV(csv string) error {
	parts := strings.Split(csv, ",")
	cols := make([]string, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s != "" {
			cols = append(cols, s)
		}
	}
	if len(cols) == 0 {
		return fmt.Errorf("AB_COLUMNS contained no valid column names")
	}
	ColumnOrder = cols
	return nil
}

// NextColumn returns the next column after cur in ColumnOrder.
func NextColumn(cur string) (string, error) {
	for i, c := range ColumnOrder {
		if c == cur {
			if i+1 < len(ColumnOrder) {
				return ColumnOrder[i+1], nil
			}
			return "", fmt.Errorf("already in last column %q", cur)
		}
	}
	return "", fmt.Errorf("unknown current column %q", cur)
}

// PrevColumn returns the previous column before cur in ColumnOrder.
func PrevColumn(cur string) (string, error) {
	for i, c := range ColumnOrder {
		if c == cur {
			if i-1 >= 0 {
				return ColumnOrder[i-1], nil
			}
			return "", fmt.Errorf("already in first column %q", cur)
		}
	}
	return "", fmt.Errorf("unknown current column %q", cur)
}
