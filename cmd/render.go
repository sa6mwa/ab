package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/sa6mwa/ab/internal/az"
	"github.com/sa6mwa/ab/internal/term"
	"github.com/sa6mwa/ab/internal/util"
)

func renderWorkItem(title string, wi *az.WorkItem) error {
	if wi == nil {
		return fmt.Errorf("no work-item to render")
	}
	typ := util.FieldString(wi.Fields, "System.WorkItemType")
	state := util.FieldString(wi.Fields, "System.State")
	assigned := util.FieldString(wi.Fields, "System.AssignedTo")
	if assigned == "" {
		if v, ok := wi.Fields["System.AssignedTo"]; ok {
			if m, ok := v.(map[string]any); ok {
				if dn, ok := m["displayName"].(string); ok {
					assigned = dn
				}
			}
		}
	}
	t := util.FieldString(wi.Fields, "System.Title")
	tags := util.FieldString(wi.Fields, "System.Tags")
	tagsOut := formatTags(tags)
	_, kanban := util.FindKanbanColumn(wi.Fields)
	severity := ""
	if typ == "Bug" {
		severity = util.FieldString(wi.Fields, "Microsoft.VSTS.Common.Severity")
	}
	url := wi.URL
	// Build details, conditionally including Severity for Bugs
	lines := []string{
		fmt.Sprintf("- ID: %d", wi.ID),
		fmt.Sprintf("- Type: %s", typ),
		fmt.Sprintf("- State: %s", state),
		fmt.Sprintf("- Title: %s", escapePipes(t)),
		fmt.Sprintf("- Assigned To: %s", assigned),
	}
	if typ == "Bug" && strings.TrimSpace(severity) != "" {
		lines = append(lines, fmt.Sprintf("- Severity: %s", severity))
	}
	lines = append(lines,
		fmt.Sprintf("- Kanban Column: %s", kanban),
		fmt.Sprintf("- Tags: %s", tagsOut),
		fmt.Sprintf("- URL: %s", url),
	)
	md := fmt.Sprintf(`# %s

%s
`, title, strings.Join(lines, "\n"))
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(term.DetectWidth()),
		glamour.WithPreservedNewLines(),
	)
	if err != nil {
		return err
	}
	out, err := r.Render(md)
	if err != nil {
		return err
	}
	fmt.Print(out)
	return nil
}

func escapePipes(s string) string { return strings.ReplaceAll(s, "|", "\\|") }

func formatTags(tags string) string {
	if strings.TrimSpace(tags) == "" {
		return "(none)"
	}
	parts := strings.Split(tags, ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return "(none)"
	}
	return strings.Join(out, ", ")
}
