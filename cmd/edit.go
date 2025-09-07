package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	h2m "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	gmd "github.com/gomarkdown/markdown"
	"github.com/sa6mwa/ab/internal/az"
	"github.com/sa6mwa/ab/internal/board"
	"github.com/sa6mwa/ab/internal/util"
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit [id]",
	Short: "Edit a work-item (title, description, assignee, state, column)",
	Args:  cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var id string
		if len(args) == 1 {
			id = args[0]
		} else {
			// pick non-Closed work item
			picked, err := pickNonClosedID()
			if err != nil {
				return err
			}
			id = picked
		}
		// get current
		_, wi, err := az.ShowWorkItem(id)
		if err != nil {
			return err
		}
		if wi == nil {
			return fmt.Errorf("unable to inspect work item %s", id)
		}

		wtype := util.FieldString(wi.Fields, "System.WorkItemType")
		title := util.FieldString(wi.Fields, "System.Title")
		state := util.FieldString(wi.Fields, "System.State")
		assignee := assigneeDisplay(wi.Fields)
		_, curCol := util.FindKanbanColumn(wi.Fields)
		// description HTML -> markdown (naive)
		html := util.FieldString(wi.Fields, "System.Description")
		originalMD := htmlToMarkdown(html)
		descMD := originalMD

		// Build form
		heading := fmt.Sprintf("Editing %s AB#%s", wtype, id)
		hstyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")) // bright blue
		fmt.Fprintln(os.Stderr, hstyle.Render(heading))
		fmt.Fprintln(os.Stderr)
		titleInput := huh.NewInput().Title("Title").Value(&title).Validate(func(s string) error {
			if strings.TrimSpace(s) == "" {
				return fmt.Errorf("title is required")
			}
			return nil
		})

		var stateSelect *huh.Select[string]
		var severitySelect *huh.Select[string]
		var severity string
		if wtype != "User Story" {
			var stateOptions []huh.Option[string]
			if wtype == "Task" {
				for _, s := range []string{"New", "Active", "Closed"} {
					stateOptions = append(stateOptions, huh.NewOption(s, s))
				}
			} else {
				for _, s := range []string{"New", "Active", "Resolved", "Closed"} {
					stateOptions = append(stateOptions, huh.NewOption(s, s))
				}
			}
			stateSelect = huh.NewSelect[string]().Title("State").Options(stateOptions...).Value(&state)
			if wtype == "Bug" {
				// Preselect existing severity or default to Medium
				severity = util.FieldString(wi.Fields, "Microsoft.VSTS.Common.Severity")
				if strings.TrimSpace(severity) == "" {
					severity = "3 - Medium"
				}
				severitySelect = huh.NewSelect[string]().Title("Severity").Options(
					huh.NewOption("1 - Critical", "1 - Critical"),
					huh.NewOption("2 - High", "2 - High"),
					huh.NewOption("3 - Medium", "3 - Medium"),
					huh.NewOption("4 - Low", "4 - Low"),
				).Value(&severity)
			}
		}

		assigneeInput := huh.NewInput().Title("Assignee (Name Surname or email)").Description("Leave empty to unassign").Value(&assignee)

		// Column only for User Stories
		var colSelect *huh.Select[string]
		var newCol string
		if wtype == "User Story" {
			def := curCol
			if def == "" || !contains(board.ColumnOrder, def) {
				def = board.ColumnOrder[0]
			}
			newCol = def
			colSelect = huh.NewSelect[string]().Title("Kanban Column").Options(optsFrom(board.ColumnOrder)...).Value(&newCol)
		}

		descArea := huh.NewText().Title("Description (Markdown)").Value(&descMD).Lines(10)
		// Acceptance Criteria (User Story only)
		acHTML := ""
		acOriginalMD := ""
		acMD := ""
		var acArea *huh.Text
		if wtype == "User Story" {
			acHTML = util.FieldString(wi.Fields, "Microsoft.VSTS.Common.AcceptanceCriteria")
			acOriginalMD = htmlToMarkdown(acHTML)
			acMD = acOriginalMD
			acArea = huh.NewText().Title("Acceptance Criteria (Markdown)").Lines(6).Value(&acMD)
		}

		// Confirm OK/Cancel
		var proceed bool
		confirm := huh.NewConfirm().Title("Apply changes?").Value(&proceed)

		// Build form groups
		var groups []*huh.Group
		switch {
		case wtype == "User Story":
			groups = []*huh.Group{huh.NewGroup(titleInput, colSelect, assigneeInput, descArea, acArea, confirm)}
		case wtype == "Bug" && stateSelect != nil:
			// For Bug: Title, Severity, State, Assignee, Description, Confirm
			groups = []*huh.Group{huh.NewGroup(titleInput, severitySelect, stateSelect, assigneeInput, descArea, confirm)}
		case stateSelect != nil:
			groups = []*huh.Group{huh.NewGroup(titleInput, stateSelect, assigneeInput, descArea, confirm)}
		default:
			groups = []*huh.Group{huh.NewGroup(titleInput, assigneeInput, descArea, confirm)}
		}
		form := huh.NewForm(groups...)
		if err := form.Run(); err != nil {
			return err
		}
		if !proceed {
			return fmt.Errorf("cancelled")
		}

		// Prepare changes
		fields := map[string]string{}
		if title != util.FieldString(wi.Fields, "System.Title") {
			fields["System.Title"] = title
		}
		if state != util.FieldString(wi.Fields, "System.State") {
			fields["System.State"] = state
		}

		// Column change
		if wtype != "Task" && newCol != curCol && newCol != "" {
			if key, _ := util.FindKanbanColumn(wi.Fields); key != "" {
				fields[key] = newCol
			}
		}

		// Description: compare markdown-to-markdown to avoid HTML noise
		if strings.TrimSpace(descMD) != strings.TrimSpace(originalMD) {
			newHTML := markdownToHTML(descMD)
			fields["System.Description"] = newHTML
		}
		// Acceptance Criteria (only for User Story)
		if wtype == "User Story" && strings.TrimSpace(acMD) != strings.TrimSpace(acOriginalMD) {
			fields["Microsoft.VSTS.Common.AcceptanceCriteria"] = markdownToHTML(acMD)
		}

		// If assignee cleared by user, include clear in fields update
		if strings.TrimSpace(assignee) == "" && assigneeDisplay(wi.Fields) != "" {
			fields["System.AssignedTo"] = ""
		}

		// Apply updates
		var lastRaw []byte
		if len(fields) > 0 {
			raw, err := az.UpdateWorkItemFields(id, fields)
			if err != nil {
				return err
			}
			lastRaw = raw
		}
		// Severity change for Bug
		if wtype == "Bug" {
			curSev := util.FieldString(wi.Fields, "Microsoft.VSTS.Common.Severity")
			if strings.TrimSpace(severity) == "" {
				severity = "3 - Medium"
			}
			if strings.TrimSpace(severity) != strings.TrimSpace(curSev) {
				raw, err := az.UpdateWorkItemFields(id, map[string]string{"Microsoft.VSTS.Common.Severity": severity})
				if err != nil {
					return err
				}
				lastRaw = raw
			}
		}
		if assignee != assigneeDisplay(wi.Fields) && strings.TrimSpace(assignee) != "" {
			raw, err := az.UpdateWorkItemAssignee(id, assignee)
			if err != nil {
				return err
			}
			lastRaw = raw
		}
		if len(lastRaw) == 0 {
			// nothing changed
			return renderWorkItem("No changes", wi)
		}
		var updated az.WorkItem
		if err := json.Unmarshal(lastRaw, &updated); err == nil {
			return renderWorkItem("Edited", &updated)
		}
		// fallback
		return az.PrintJSON(lastRaw)
	},
}

func init() { rootCmd.AddCommand(editCmd) }

func optsFrom(values []string) []huh.Option[string] {
	out := make([]huh.Option[string], 0, len(values))
	for _, v := range values {
		out = append(out, huh.NewOption(v, v))
	}
	return out
}

// very naive HTML -> Markdown
func htmlToMarkdown(s string) string {
	if strings.TrimSpace(s) == "" {
		return ""
	}
	conv := h2m.NewConverter("", true, nil)
	md, err := conv.ConvertString(s)
	if err != nil {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(md)
}

// very naive Markdown -> HTML paragraphs
func markdownToHTML(s string) string {
	if strings.TrimSpace(s) == "" {
		return ""
	}
	// Use gomarkdown/markdown to render to HTML fragment (no html/body wrapper)
	return string(gmd.ToHTML([]byte(s), nil, nil))
}

func normalizeHTML(s string) string {
	return strings.ReplaceAll(strings.TrimSpace(s), "\n", "")
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}
