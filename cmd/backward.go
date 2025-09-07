package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sa6mwa/ab/internal/az"
	"github.com/sa6mwa/ab/internal/board"
	"github.com/sa6mwa/ab/internal/util"
	"github.com/spf13/cobra"
)

var backwardCmd = &cobra.Command{
	Use:   "backward [id]",
	Short: "Move a work-item backward one Kanban column",
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
		_, item, err := az.ShowWorkItem(id)
		if err != nil {
			return err
		}
		if item == nil {
			return fmt.Errorf("unable to inspect work item %s", id)
		}

		colField, curCol := util.FindKanbanColumn(item.Fields)
		if colField == "" || curCol == "" {
			return fmt.Errorf("kanban column field not found on work item %s", id)
		}
		prevCol, err := board.PrevColumn(curCol)
		if err != nil {
			return err
		}

		fields := map[string]string{colField: prevCol}
		raw, err := az.UpdateWorkItemFields(id, fields)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Moved column from %s to %s\n", curCol, prevCol)

		var wi az.WorkItem
		if err := json.Unmarshal(raw, &wi); err == nil {
			return renderWorkItem("Item stepped back", &wi)
		}
		return az.PrintJSON(raw)
	},
}

// order helpers centralized in internal/board

func init() { rootCmd.AddCommand(backwardCmd) }
