package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/sa6mwa/ab/internal/az"
	"github.com/spf13/cobra"
)

var bugParentID string
var bugAssignee string
var bugSeverity string

var createBugCmd = &cobra.Command{
	Use:   "bug [\"Title...\"]",
	Short: "Create a Bug under a User Story",
	Args:  cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return interactiveCreateBug()
		}
		title := args[0]
		// Resolve parent user story
		pid := strings.TrimSpace(bugParentID)
		if pid == "" {
			// Query non-Closed User Stories; honor PO order
			items, err := queryItemsWithOrder("User Story", false, poOrderGlobal)
			if err != nil {
				return err
			}
			type opt struct{ label, value string }
			opts := make([]opt, 0, len(items))
			for _, it := range items {
				idStr := strconv.Itoa(it.ID)
				t := utilField(it.Fields, "System.Title")
				if t == "" {
					t = "(no title)"
				}
				opts = append(opts, opt{label: fmt.Sprintf("%s | %s", idStr, t), value: idStr})
			}
			if len(opts) == 0 {
				return fmt.Errorf("no User Stories available to select as parent")
			}
			sel := huh.NewSelect[string]().Title("Pick parent User Story")
			var optItems []huh.Option[string]
			for _, o := range opts {
				optItems = append(optItems, huh.NewOption[string](o.label, o.value))
			}
			sel = sel.Options(optItems...)
			var chosen string
			sel.Value(&chosen)
			if err := huh.NewForm(huh.NewGroup(sel)).Run(); err != nil {
				return err
			}
			if strings.TrimSpace(chosen) == "" {
				return fmt.Errorf("no parent selected")
			}
			pid = chosen
		} else {
			// Validate provided parent is a User Story
			if _, wi, err := az.ShowWorkItem(pid); err != nil {
				return fmt.Errorf("validate parent: %w", err)
			} else if wi == nil || utilField(wi.Fields, "System.WorkItemType") != "User Story" {
				return fmt.Errorf("parent %s is not a User Story", pid)
			}
		}

		fields := map[string]string{"System.State": "New"}
		if strings.TrimSpace(bugAssignee) != "" {
			at := strings.TrimSpace(bugAssignee)
			if at == "@me" {
				me, err := az.CurrentUserUPN()
				if err != nil {
					return fmt.Errorf("get current user: %w", err)
				}
				fields["System.AssignedTo"] = me
			} else {
				fields["System.AssignedTo"] = at
			}
		}
		// Severity via flag (defaults to 3 - Medium if unspecified)
		if lbl, err := resolveSeverityLabel(bugSeverity); err != nil {
			return err
		} else if lbl != "" {
			fields["Microsoft.VSTS.Common.Severity"] = lbl
		} else {
			fields["Microsoft.VSTS.Common.Severity"] = "3 - Medium"
		}
		raw, err := az.CreateWorkItem("Bug", title, fields, "")
		if err != nil {
			return err
		}
		var wi az.WorkItem
		if err := json.Unmarshal(raw, &wi); err != nil {
			return az.PrintJSON(raw)
		}
		// Add parent relation (child -> parent)
		if _, err := az.AddWorkItemRelation(strconv.Itoa(wi.ID), "parent", pid); err != nil {
			return fmt.Errorf("created bug %d but failed to add parent relation to %s: %w", wi.ID, pid, err)
		}
		fmt.Fprintf(os.Stderr, "Linked AB#%d as child of AB#%s\n", wi.ID, pid)
		return renderWorkItem("Bug Created", &wi)
	},
}

func init() {
	createCmd.AddCommand(createBugCmd)
	createBugCmd.Flags().StringVarP(&bugParentID, "parent", "p", "", "Parent User Story ID (required). If omitted, shows a picker.")
	createBugCmd.Flags().StringVarP(&bugAssignee, "assignee", "a", "", "Assignee (use @me for yourself)")
	createBugCmd.Flags().StringVar(&bugSeverity, "severity", "", "Severity 1|2|3|4 (1-Critical, 2-High, 3-Medium, 4-Low)")
}

func interactiveCreateBug() error {
	pid := strings.TrimSpace(bugParentID)
	if pid == "" {
		items, err := queryItemsWithOrder("User Story", false, poOrderGlobal)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return fmt.Errorf("no User Stories available to select as parent")
		}
		var options []huh.Option[string]
		for _, it := range items {
			idStr := strconv.Itoa(it.ID)
			t := utilField(it.Fields, "System.Title")
			if t == "" {
				t = "(no title)"
			}
			options = append(options, huh.NewOption(idStr+" | "+t, idStr))
		}
		var chosen string
		if err := huh.NewForm(huh.NewGroup(huh.NewSelect[string]().Title("Pick parent User Story").Options(options...).Value(&chosen))).Run(); err != nil {
			return err
		}
		if strings.TrimSpace(chosen) == "" {
			return fmt.Errorf("no parent selected")
		}
		pid = chosen
	}

	var title, assignee, state, descMD, severity string
	state = "New"
	if lbl, err := resolveSeverityLabel(bugSeverity); err == nil && lbl != "" {
		severity = lbl
	} else {
		severity = "3 - Medium"
	}
	// prefill assignee from flag
	if strings.TrimSpace(bugAssignee) == "@me" {
		if me, err := az.CurrentUserUPN(); err == nil {
			assignee = me
		}
	} else if strings.TrimSpace(bugAssignee) != "" {
		assignee = strings.TrimSpace(bugAssignee)
	}
	// Heading showing parent context
	heading := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")).Render(fmt.Sprintf("Adding Bug to %s: %s", pid, parentTitleByID(pid)))
	fmt.Fprintln(os.Stderr, heading)
	fmt.Fprintln(os.Stderr)
	var proceed bool
	f := huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Title").Value(&title).Validate(func(s string) error {
			if strings.TrimSpace(s) == "" {
				return fmt.Errorf("title is required")
			}
			return nil
		}),
		huh.NewSelect[string]().Title("Severity").Options(
			huh.NewOption("1 - Critical", "1 - Critical"),
			huh.NewOption("2 - High", "2 - High"),
			huh.NewOption("3 - Medium", "3 - Medium"),
			huh.NewOption("4 - Low", "4 - Low"),
		).Value(&severity),
		// Use Bug states (Agile): New, Active, Resolved, Closed
		huh.NewSelect[string]().Title("State").Options(
			huh.NewOption("New", "New"), huh.NewOption("Active", "Active"), huh.NewOption("Resolved", "Resolved"), huh.NewOption("Closed", "Closed"),
		).Value(&state),
		huh.NewInput().Title("Assignee (Name or email)").Value(&assignee),
		huh.NewText().Title("Description (Markdown)").Lines(8).Value(&descMD),
		huh.NewConfirm().Title("Create Bug?").Value(&proceed),
	))
	if err := f.Run(); err != nil {
		return err
	}
	if !proceed {
		return fmt.Errorf("cancelled")
	}
	fields := map[string]string{"System.State": state}
	if strings.TrimSpace(severity) != "" {
		fields["Microsoft.VSTS.Common.Severity"] = severity
	}
	if strings.TrimSpace(assignee) != "" {
		fields["System.AssignedTo"] = assignee
	}
	if strings.TrimSpace(descMD) != "" {
		fields["System.Description"] = markdownToHTML(descMD)
	}
	raw, err := az.CreateWorkItem("Bug", title, fields, "")
	if err != nil {
		return err
	}
	var wi az.WorkItem
	if err := json.Unmarshal(raw, &wi); err != nil {
		return az.PrintJSON(raw)
	}
	if _, err := az.AddWorkItemRelation(strconv.Itoa(wi.ID), "parent", pid); err != nil {
		return fmt.Errorf("created bug %d but failed to add parent relation to %s: %w", wi.ID, pid, err)
	}
	return renderWorkItem("Bug Created", &wi)
}

// resolveSeverityLabel maps a numeric flag (1-4) to the display label expected by Azure Boards.
func resolveSeverityLabel(v string) (string, error) {
	vv := strings.TrimSpace(v)
	switch vv {
	case "":
		return "", nil
	case "1":
		return "1 - Critical", nil
	case "2":
		return "2 - High", nil
	case "3":
		return "3 - Medium", nil
	case "4":
		return "4 - Low", nil
	default:
		return "", fmt.Errorf("invalid --severity value %q (use 1,2,3,4)", v)
	}
}
