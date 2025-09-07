package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/sa6mwa/ab/internal/az"
	"github.com/spf13/cobra"
)

var workonCmd = &cobra.Command{
	Use:   "workon [id]",
	Short: "Assign to me and move to Active",
	Args:  cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var id string
		if len(args) == 1 {
			id = args[0]
		} else {
			var err error
			id, err = pickNonClosedID()
			if err != nil {
				return err
			}
		}
		me, err := az.CurrentUserUPN()
		if err != nil {
			return fmt.Errorf("get current user: %w", err)
		}
		raw, err := az.UpdateWorkItemFields(id, map[string]string{
			"System.AssignedTo": me,
			"System.State":      "Active",
		})
		if err != nil {
			return err
		}
		var wi az.WorkItem
		if err := json.Unmarshal(raw, &wi); err == nil {
			return renderWorkItem("Working On", &wi)
		}
		return az.PrintJSON(raw)
	},
}

// pickWorkItem lets the user pick an item; if onlyMine is true, restrict to items assigned to @me
// no interactive selection; ID is required

func init() {
	rootCmd.AddCommand(workonCmd)
}
