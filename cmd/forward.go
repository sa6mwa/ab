package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/sa6mwa/ab/internal/az"
	"github.com/sa6mwa/ab/internal/board"
	"github.com/sa6mwa/ab/internal/util"
	"github.com/spf13/cobra"
)

var forwardCmd = &cobra.Command{
	Use:   "forward [id]",
	Short: "Push a work-item forward",
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

		// Determine dynamic Kanban column field and current column
		colField, curCol := util.FindKanbanColumn(item.Fields)
		if colField == "" || curCol == "" {
			return fmt.Errorf("kanban column field not found on work item %s", id)
		}
		nextCol, err := board.NextColumn(curCol)
		if err != nil {
			return err
		}

		fields := map[string]string{colField: nextCol}

		raw, err := az.UpdateWorkItemFields(id, fields)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Moved column from %s to %s\n", curCol, nextCol)
		// Pretty render updated work item
		var wi az.WorkItem
		if err := json.Unmarshal(raw, &wi); err == nil {
			return renderWorkItem("Item pushed forward", &wi)
		}
		return az.PrintJSON(raw)
	},
}

func idString(wi *az.WorkItem) string { return strconv.Itoa(wi.ID) }

// findKanbanColumn finds the dynamic WEF_*_Kanban.Column field and value.
// no tag manipulation in forward

func init() { rootCmd.AddCommand(forwardCmd) }
