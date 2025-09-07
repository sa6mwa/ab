package cmd

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/huh"
	"github.com/sa6mwa/ab/internal/az"
	"github.com/sa6mwa/ab/internal/term"
	"github.com/sa6mwa/ab/internal/util"
	"github.com/spf13/cobra"
)

var showIncludeAll bool
var showOutputPath string
var showOutputPick bool

var showCmd = &cobra.Command{
	Use:   "show [id]",
	Short: "Show a work-item and its details",
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

		_, wi, err := az.ShowWorkItem(id)
		if err != nil {
			return err
		}
		if wi == nil {
			return fmt.Errorf("unable to inspect work item %s", id)
		}

		// If User Story, prefetch children to include at bottom of single document
		var children []queryItem
		if util.FieldString(wi.Fields, "System.WorkItemType") == "User Story" {
			children, err = queryItemsByParent(fmt.Sprintf("%d", wi.ID), showIncludeAll)
			if err != nil {
				return err
			}
		}

		// Build Markdown document with compact pseudo-headings
		var b bytes.Buffer
		title := util.FieldString(wi.Fields, "System.Title")
		wtype := util.FieldString(wi.Fields, "System.WorkItemType")
		state := util.FieldString(wi.Fields, "System.State")

		// Created By display name
		createdBy := createdByDisplay(wi.Fields)
		if createdBy == "" {
			createdBy = "(unknown)"
		}

		// Assignee display with NIL if none
		assignee := assigneeDisplay(wi.Fields)
		if strings.TrimSpace(assignee) == "" {
			assignee = "NIL"
		}

		// Column (User Story only)
		_, col := util.FindKanbanColumn(wi.Fields)

		// Description and Acceptance Criteria converted from HTML -> Markdown
		descHTML := util.FieldString(wi.Fields, "System.Description")
		descMD := htmlToMarkdown(descHTML)
		acMD := ""
		if wtype == "User Story" {
			acHTML := util.FieldString(wi.Fields, "Microsoft.VSTS.Common.AcceptanceCriteria")
			acMD = htmlToMarkdown(acHTML)
		}

		// Compose markdown
		fmt.Fprintf(&b, "# %s AB#%d\n\n", wtype, wi.ID)
		fmt.Fprintf(&b, "**Title:**  \n%s\n\n", strings.TrimSpace(title))
		if wtype == "Bug" {
			sev := util.FieldString(wi.Fields, "Microsoft.VSTS.Common.Severity")
			if strings.TrimSpace(sev) == "" {
				sev = "NIL"
			}
			fmt.Fprintf(&b, "**Severity:**  \n%s\n\n", sev)
		}
		fmt.Fprintf(&b, "**Created By:**  \n%s\n\n", createdBy)
		fmt.Fprintf(&b, "**Assignee:**  \n%s\n\n", assignee)
		if wtype == "User Story" {
			if strings.TrimSpace(col) == "" {
				fmt.Fprintf(&b, "**Column:**  \nNIL\n\n")
			} else {
				fmt.Fprintf(&b, "**Column:**  \n%s\n\n", col)
			}
		}
		fmt.Fprintf(&b, "**State:**  \n%s\n\n", state)
		if strings.TrimSpace(descMD) == "" {
			fmt.Fprintf(&b, "**Description:**  \nNIL\n\n")
		} else {
			fmt.Fprintf(&b, "**Description:**  \n%s\n\n", descMD)
		}
		if wtype == "User Story" {
			if strings.TrimSpace(acMD) == "" {
				fmt.Fprintf(&b, "**Acceptance Criteria:**  \nNIL\n\n")
			} else {
				fmt.Fprintf(&b, "**Acceptance Criteria:**  \n%s\n\n", acMD)
			}
		}

		// Children section (User Story only), appended within same document
		if wtype == "User Story" {
			fmt.Fprintf(&b, "# Children\n\n")
			if len(children) == 0 {
				b.WriteString("No work-items found.\n")
			} else {
				sort.Slice(children, func(i, j int) bool { return children[i].ID > children[j].ID })
				b.WriteString("| ID | Type | State | Assignee | Title |\n")
				b.WriteString("|---:|:-----|:------|:---------|:------|\n")
				for _, c := range children {
					t := util.FieldString(c.Fields, "System.WorkItemType")
					s := util.FieldString(c.Fields, "System.State")
					ass := assigneeDisplay(c.Fields)
					title := util.FieldString(c.Fields, "System.Title")
					title = strings.ReplaceAll(title, "|", "\\|")
					fmt.Fprintf(&b, "| %d | %s | %s | %s | %s |\n", c.ID, t, s, ass, title)
				}
			}
			b.WriteString("\n")
		}

		// Render document
		r, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(term.DetectWidth()),
			glamour.WithPreservedNewLines(),
		)
		if err != nil {
			return err
		}
		md := b.String()
		out, err := r.Render(md)
		if err != nil {
			return err
		}
		fmt.Print(out)
		// After printing, handle saving to file if requested
		if showOutputPick {
			def := fmt.Sprintf("ab%d.md", wi.ID)
			var path string = def
			inp := huh.NewInput().Title("Save markdown as").Value(&path)
			if err := huh.NewForm(huh.NewGroup(inp)).Run(); err != nil {
				return err
			}
			if strings.TrimSpace(path) != "" {
				if xp, err := util.ExpandTilde(path); err == nil {
					path = xp
				}
				if err := os.WriteFile(path, []byte(md), 0644); err != nil {
					return fmt.Errorf("save markdown: %w", err)
				}
				fmt.Fprintf(os.Stderr, "Saved markdown to %s\n", path)
			}
		} else if strings.TrimSpace(showOutputPath) != "" {
			xp, err := util.ExpandTilde(showOutputPath)
			if err == nil {
				showOutputPath = xp
			}
			if err := os.WriteFile(showOutputPath, []byte(md), 0644); err != nil {
				return fmt.Errorf("save markdown: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Saved markdown to %s\n", showOutputPath)
		}
		return nil
	},
}

func init() { rootCmd.AddCommand(showCmd) }
func init() {
	showCmd.Flags().StringVarP(&showOutputPath, "output", "o", "", "Write generated Markdown to file")
	showCmd.Flags().BoolVarP(&showOutputPick, "output-pick", "O", false, "Pick output file path interactively")
}
func init() {
	showCmd.Flags().BoolVarP(&showIncludeAll, "all", "a", false, "Include Closed children in the list")
}

// createdByDisplay extracts System.CreatedBy.displayName when present
func createdByDisplay(fields map[string]interface{}) string {
	if v, ok := fields["System.CreatedBy"]; ok {
		switch t := v.(type) {
		case string:
			return t
		case map[string]any:
			if dn, ok := t["displayName"].(string); ok {
				return dn
			}
		}
	}
	return ""
}
