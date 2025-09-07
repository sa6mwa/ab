package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/sa6mwa/ab/internal/az"
	"github.com/sa6mwa/ab/internal/board"
	"github.com/sa6mwa/ab/internal/util"
	"github.com/spf13/cobra"
)

var assignTo string

var createStoryCmd = &cobra.Command{
	Use:   "story [\"Title...\"]",
	Short: "Create a User Story",
	Args:  cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return interactiveCreateStory()
		}
		title := args[0]
		fields := map[string]string{"System.State": "New"}
		at := strings.TrimSpace(assignTo)
		if at != "" {
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
		raw, err := az.CreateWorkItem("User Story", title, fields, "")
		if err != nil {
			return err
		}
		var wi az.WorkItem
		if err := json.Unmarshal(raw, &wi); err != nil {
			return az.PrintJSON(raw)
		}
		return renderWorkItem("User Story Created", &wi)
	},
}

func init() {
	createCmd.AddCommand(createStoryCmd)
	createStoryCmd.Flags().StringVarP(&assignTo, "assign", "a", "", "Assign to user (use @me for yourself)")
}

func interactiveCreateStory() error {
	var title, assignee, col, descMD, acMD string
	col = board.ColumnOrder[0]
	// Prefill assignee if provided via -a, resolving @me to current UPN
	if strings.TrimSpace(assignTo) == "@me" {
		if me, err := az.CurrentUserUPN(); err == nil {
			assignee = me
		}
	} else if strings.TrimSpace(assignTo) != "" {
		assignee = strings.TrimSpace(assignTo)
	}
	var proceed bool
	form := huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Title").Value(&title).Validate(func(s string) error {
			if strings.TrimSpace(s) == "" {
				return fmt.Errorf("title is required")
			}
			return nil
		}),
		huh.NewSelect[string]().Title("Kanban Column").Options(optsFrom(board.ColumnOrder)...).Value(&col),
		huh.NewInput().Title("Assignee (Name or email)").Value(&assignee),
		huh.NewText().Title("Description (Markdown)").Lines(8).Value(&descMD),
		huh.NewText().Title("Acceptance Criteria (Markdown)").Lines(6).Value(&acMD),
		huh.NewConfirm().Title("Create User Story?").Value(&proceed),
	))
	if err := form.Run(); err != nil {
		return err
	}
	if !proceed {
		return fmt.Errorf("cancelled")
	}
	fields := map[string]string{"System.State": "New"}
	if strings.TrimSpace(assignee) != "" {
		fields["System.AssignedTo"] = assignee
	}
	if strings.TrimSpace(descMD) != "" {
		fields["System.Description"] = markdownToHTML(descMD)
	}
	if strings.TrimSpace(acMD) != "" {
		fields["Microsoft.VSTS.Common.AcceptanceCriteria"] = markdownToHTML(acMD)
	}
	raw, err := az.CreateWorkItem("User Story", title, fields, "")
	if err != nil {
		return err
	}
	var wi az.WorkItem
	if err := json.Unmarshal(raw, &wi); err == nil {
		// If user selected a starting column, update and render the updated item
		if key, _ := util.FindKanbanColumn(wi.Fields); key != "" && strings.TrimSpace(col) != "" {
			if uraw, err := az.UpdateWorkItemFields(strconv.Itoa(wi.ID), map[string]string{key: col}); err == nil {
				var updated az.WorkItem
				if json.Unmarshal(uraw, &updated) == nil {
					return renderWorkItem("User Story Created", &updated)
				}
				// Fallback to printing raw updated JSON if shape changed
				return az.PrintJSON(uraw)
			} else {
				return err
			}
		}
		// No column change requested; render created item
		return renderWorkItem("User Story Created", &wi)
	}
	return az.PrintJSON(raw)
}
