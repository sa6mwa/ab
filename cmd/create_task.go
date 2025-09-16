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

var parentID string
var taskAssignee string

var createTaskCmd = &cobra.Command{
	Use:   "task [\"Title...\"]",
	Short: "Create a Task under a User Story",
	Args:  cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return interactiveCreateTask()
		}
		title := args[0]
		// Resolve parent user story
		pid := strings.TrimSpace(parentID)
		if pid == "" {
			// Query non-Closed User Stories via WIQL (single call)
			items, err := queryItemsWithOrder("User Story", false, poOrderGlobal)
			if err != nil {
				return err
			}
			type opt struct{ label, value string }
			opts := make([]opt, 0, len(items))
			for _, it := range items {
				// Only non-Closed; baseListWIQL already filtered
				idStr := strconv.Itoa(it.ID)
				title := utilField(it.Fields, "System.Title")
				if title == "" {
					title = "(no title)"
				}
				opts = append(opts, opt{label: fmt.Sprintf("%s | %s", idStr, title), value: idStr})
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
			form := huh.NewForm(huh.NewGroup(sel))
			if err := form.Run(); err != nil {
				return err
			}
			if strings.TrimSpace(chosen) == "" {
				return fmt.Errorf("no parent selected")
			}
			pid = chosen
		}

		// If parent provided via flag, validate it's a User Story
		if parentID != "" {
			if _, wi, err := az.ShowWorkItem(pid); err != nil {
				return fmt.Errorf("validate parent: %w", err)
			} else if wi == nil || utilField(wi.Fields, "System.WorkItemType") != "User Story" {
				return fmt.Errorf("parent %s is not a User Story", pid)
			}
		}

		// Create without setting state to avoid create-time state restrictions
		fields := map[string]string{}
		if strings.TrimSpace(taskAssignee) != "" {
			at := strings.TrimSpace(taskAssignee)
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
		raw, err := az.CreateWorkItem("Task", title, fields, "")
		if err != nil {
			return err
		}
		var wi az.WorkItem
		if err := json.Unmarshal(raw, &wi); err != nil {
			return az.PrintJSON(raw)
		}
		// Add parent relation (child -> parent)
		if _, err := az.AddWorkItemRelation(strconv.Itoa(wi.ID), "parent", pid); err != nil {
			return fmt.Errorf("created task %d but failed to add parent relation to %s: %w", wi.ID, pid, err)
		}
		fmt.Fprintf(os.Stderr, "Linked AB#%d as child of AB#%s\n", wi.ID, pid)
		return renderWorkItem("Task Created", &wi)
	},
}

// utilField safely extracts a string from a generic fields map
func utilField(fields map[string]any, key string) string {
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

func init() {
	createCmd.AddCommand(createTaskCmd)
	createTaskCmd.Flags().StringVarP(&parentID, "parent", "p", "", "Parent User Story ID (required). If omitted, shows a picker.")
	createTaskCmd.Flags().StringVarP(&taskAssignee, "assignee", "a", "", "Assignee (use @me for yourself)")
}

func interactiveCreateTask() error {
	pid := strings.TrimSpace(parentID)
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
	var title, assignee, state, descMD string
	state = "New"
	// prefill assignee from flag
	if strings.TrimSpace(taskAssignee) == "@me" {
		if me, err := az.CurrentUserUPN(); err == nil {
			assignee = me
		}
	} else if strings.TrimSpace(taskAssignee) != "" {
		assignee = strings.TrimSpace(taskAssignee)
	}
	// Heading showing parent context
	heading := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")).Render(fmt.Sprintf("Adding Task to %s: %s", pid, parentTitleByID(pid)))
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
		huh.NewSelect[string]().Title("State").Options(
			huh.NewOption("New", "New"), huh.NewOption("Active", "Active"), huh.NewOption("Closed", "Closed"),
		).Value(&state),
		huh.NewInput().Title("Assignee (Name or email)").Value(&assignee),
		huh.NewText().Title("Description (Markdown)").Lines(8).Value(&descMD),
		huh.NewConfirm().Title("Create Task?").Value(&proceed),
	))
	if err := f.Run(); err != nil {
		return err
	}
	if !proceed {
		return fmt.Errorf("cancelled")
	}
	// Create without state; set it with a follow-up update
	fields := map[string]string{}
	if strings.TrimSpace(assignee) != "" {
		fields["System.AssignedTo"] = assignee
	}
	if strings.TrimSpace(descMD) != "" {
		fields["System.Description"] = markdownToHTML(descMD)
	}
	raw, err := az.CreateWorkItem("Task", title, fields, "")
	if err != nil {
		return err
	}
	var wi az.WorkItem
	if err := json.Unmarshal(raw, &wi); err != nil {
		return az.PrintJSON(raw)
	}
	if _, err := az.AddWorkItemRelation(strconv.Itoa(wi.ID), "parent", pid); err != nil {
		return fmt.Errorf("created task %d but failed to add parent relation to %s: %w", wi.ID, pid, err)
	}
	// Update state after creation only if not default "New"
	if strings.TrimSpace(state) != "" && strings.TrimSpace(state) != "New" {
		if raw, err := az.UpdateWorkItemFields(strconv.Itoa(wi.ID), map[string]string{"System.State": state}); err == nil {
			var updated az.WorkItem
			if json.Unmarshal(raw, &updated) == nil {
				return renderWorkItem("Task Created", &updated)
			}
		}
	}
	return renderWorkItem("Task Created", &wi)
}

func parentTitleByID(id string) string {
	_, wi, err := az.ShowWorkItem(id)
	if err != nil || wi == nil {
		return ""
	}
	return utilField(wi.Fields, "System.Title")
}
