package util

import "strings"

// FieldString safely extracts a string field from a work item fields map.
func FieldString(fields map[string]interface{}, key string) string {
	if fields == nil {
		return ""
	}
	if v, ok := fields[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// FindKanbanColumn locates the dynamic Kanban column field and value.
// It searches for a field key matching the pattern: WEF_*_Kanban.Column
func FindKanbanColumn(fields map[string]interface{}) (name string, value string) {
	if fields == nil {
		return "", ""
	}
	for k, v := range fields {
		if strings.HasPrefix(k, "WEF_") && strings.Contains(k, "Kanban.Column") {
			if s, ok := v.(string); ok {
				return k, s
			}
		}
	}
	return "", ""
}
