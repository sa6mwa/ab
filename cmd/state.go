package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/sa6mwa/ab/internal/az"
	"github.com/sa6mwa/ab/internal/util"
	"github.com/spf13/cobra"
)

var resolveCmd = &cobra.Command{
	Use:   "resolve [id]",
	Short: "Set work-item state to Resolved",
	Args:  cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var ids []string
		if len(args) == 1 {
			ids = []string{args[0]}
		} else {
			var err error
			ids, err = pickNonClosedIDs()
			if err != nil {
				return err
			}
		}
		for _, id := range ids {
			_, cur, err := az.ShowWorkItem(id)
			if err != nil {
				return err
			}
			if cur == nil {
				return fmt.Errorf("unable to inspect work item %s", id)
			}
			target := "Resolved"
			if util.FieldString(cur.Fields, "System.WorkItemType") == "Task" {
				target = "Closed"
			}
			raw, err := az.UpdateWorkItemFields(id, map[string]string{"System.State": target})
			if err != nil {
				return err
			}
			var wi az.WorkItem
			if err := json.Unmarshal(raw, &wi); err == nil {
				if err := renderWorkItem("Resolved", &wi); err != nil {
					return err
				}
			} else {
				if err := az.PrintJSON(raw); err != nil {
					return err
				}
			}
		}
		return nil
	},
}

var renewCmd = &cobra.Command{
	Use:   "renew [id]",
	Short: "Set work-item state to New",
	Args:  cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var ids []string
		if len(args) == 1 {
			ids = []string{args[0]}
		} else {
			var err error
			ids, err = pickNonClosedIDs()
			if err != nil {
				return err
			}
		}
		for _, id := range ids {
			raw, err := az.UpdateWorkItemFields(id, map[string]string{"System.State": "New"})
			if err != nil {
				return err
			}
			var wi az.WorkItem
			if err := json.Unmarshal(raw, &wi); err == nil {
				if err := renderWorkItem("Renewed", &wi); err != nil {
					return err
				}
			} else {
				if err := az.PrintJSON(raw); err != nil {
					return err
				}
			}
		}
		return nil
	},
}

var closeCmd = &cobra.Command{
	Use:   "close [id]",
	Short: "Set work-item state to Closed",
	Args:  cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var ids []string
		if len(args) == 1 {
			ids = []string{args[0]}
		} else {
			var err error
			ids, err = pickNonClosedIDs()
			if err != nil {
				return err
			}
		}
		for _, id := range ids {
			raw, err := az.UpdateWorkItemFields(id, map[string]string{"System.State": "Closed"})
			if err != nil {
				return err
			}
			var wi az.WorkItem
			if err := json.Unmarshal(raw, &wi); err == nil {
				if err := renderWorkItem("Closed", &wi); err != nil {
					return err
				}
			} else {
				if err := az.PrintJSON(raw); err != nil {
					return err
				}
			}
		}
		return nil
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete [id]",
	Short: "Delete a work-item",
	Args:  cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var ids []string
		if len(args) == 1 {
			ids = []string{args[0]}
		} else {
			var err error
			ids, err = pickNonClosedIDs()
			if err != nil {
				return err
			}
		}
		for _, id := range ids {
			raw, err := az.DeleteWorkItem(id)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "Deleted AB#%s\n", id)
			if err := az.PrintJSON(raw); err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(resolveCmd)
	rootCmd.AddCommand(renewCmd)
	rootCmd.AddCommand(closeCmd)
	rootCmd.AddCommand(deleteCmd)
}
