package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/huh"
	"github.com/sa6mwa/ab/internal/az"
	"github.com/sa6mwa/ab/internal/term"
	"github.com/sa6mwa/ab/internal/util"
	"github.com/spf13/cobra"
)

var includeAll bool
var listOutputPath string
var listOutputPick bool

var listCmd = &cobra.Command{
	Use:   "list [parentID]",
	Short: "List work-items",
	Long:  "List non-Closed work-items by default. Use -a/--all to include Closed. If a parent ID is provided, lists children of that work-item.",
	Args:  cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			// list children of the given parent work-item
			items, err := queryItemsByParent(args[0], includeAll)
			if err != nil {
				return err
			}
			// fetch parent for header
			_, parent, err := az.ShowWorkItem(args[0])
			if err != nil {
				return err
			}
			md, err := renderChildren(parent, items)
			if err != nil {
				return err
			}
			// After printing, optionally save
			if listOutputPick {
				def := fmt.Sprintf("ab%s.md", args[0])
				var path string = def
				if err := huhSavePath(&path); err != nil {
					return err
				}
				if strings.TrimSpace(path) != "" {
					if err := os.WriteFile(path, []byte(md), 0644); err != nil {
						return err
					}
					fmt.Fprintf(os.Stderr, "Saved markdown to %s\n", path)
				}
			} else if strings.TrimSpace(listOutputPath) != "" {
				if err := os.WriteFile(listOutputPath, []byte(md), 0644); err != nil {
					return err
				}
				fmt.Fprintf(os.Stderr, "Saved markdown to %s\n", listOutputPath)
			}
			return nil
		}
		// default listing
		usePO := poOrderGlobal
		var items []queryItem
		var err error
		if usePO {
			items, err = queryPOOrdered(includeAll)
		} else {
			items, err = queryItems("")
		}
		if err != nil {
			return err
		}
		md, err := renderItems(items)
		if err != nil {
			return err
		}
		if listOutputPick {
			def := "ab-list.md"
			var path string = def
			if err := huhSavePath(&path); err != nil {
				return err
			}
			if strings.TrimSpace(path) != "" {
				if err := os.WriteFile(path, []byte(md), 0644); err != nil {
					return err
				}
				fmt.Fprintf(os.Stderr, "Saved markdown to %s\n", path)
			}
		} else if strings.TrimSpace(listOutputPath) != "" {
			if err := os.WriteFile(listOutputPath, []byte(md), 0644); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Saved markdown to %s\n", listOutputPath)
		}
		return nil
	},
}

// (top-level baseListWIQL removed; see single definition below)

var tasksCmd = &cobra.Command{
	Use:   "tasks",
	Short: "List Tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		items, err := queryItems("Task")
		if err != nil {
			return err
		}
		_, err = renderItemsTypeLess(items, "Tasks")
		return err
	},
}

var storiesCmd = &cobra.Command{
	Use:   "stories",
	Short: "List User Stories",
	RunE: func(cmd *cobra.Command, args []string) error {
		items, err := queryItems("User Story")
		if err != nil {
			return err
		}
		_, err = renderItemsTypeLess(items, "User Stories")
		return err
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.PersistentFlags().BoolVarP(&includeAll, "all", "a", false, "Include Closed items")
	listCmd.PersistentFlags().StringVarP(&listOutputPath, "output", "o", "", "Write generated Markdown to file")
	listCmd.PersistentFlags().BoolVarP(&listOutputPick, "output-pick", "O", false, "Pick output file path interactively")
	listCmd.AddCommand(tasksCmd)
	listCmd.AddCommand(storiesCmd)
}

type queryItem struct {
	ID     int                    `json:"id"`
	Fields map[string]interface{} `json:"fields"`
}

// queryItems runs a WIQL selecting needed fields and returns items directly.
func queryItems(typeFilter string) ([]queryItem, error) {
	wiql := baseListWIQL(typeFilter)
	return queryItemsByWIQL(wiql)
}

// queryItemsWithOrder returns items honoring PO order when requested.
func queryItemsWithOrder(typeFilter string, includeClosed, usePO bool) ([]queryItem, error) {
	if !usePO {
		return queryItemsByWIQL(baseListWIQLWith(includeClosed, typeFilter))
	}
	// PO order requested
	// For specific types, if Story or Bug, order by StackRank; else by ChangedDate
	if typeFilter == "User Story" || typeFilter == "Bug" {
		where := []string{}
		if !includeClosed {
			where = append(where, "[System.State] <> 'Closed'")
		}
		where = append(where, fmt.Sprintf("[System.WorkItemType] = '%s'", typeFilter))
		wiql := "SELECT [System.Id], [System.Title], [System.State], [System.WorkItemType], [System.AssignedTo], [Microsoft.VSTS.Common.StackRank] FROM WorkItems"
		wiql += " WHERE " + strings.Join(where, " AND ")
		wiql += " ORDER BY [Microsoft.VSTS.Common.StackRank] ASC, [System.ChangedDate] DESC"
		return queryItemsByWIQL(wiql)
	}
	// No type filter: combine Story+Bug by StackRank, others by date
	if typeFilter == "" {
		return queryPOOrdered(includeClosed)
	}
	// Other single types: default ChangedDate ordering
	return queryItemsByWIQL(baseListWIQLWith(includeClosed, typeFilter))
}

// queryPOOrdered returns Stories/Bugs ordered by StackRank then others by ChangedDate.
func queryPOOrdered(includeClosed bool) ([]queryItem, error) {
	// First: User Story + Bug by StackRank ASC, then ChangedDate DESC
	var where1 []string
	if !includeClosed {
		where1 = append(where1, "[System.State] <> 'Closed'")
	}
	where1 = append(where1, "[System.WorkItemType] IN ('User Story','Bug')")
	wiql1 := "SELECT [System.Id], [System.Title], [System.State], [System.WorkItemType], [System.AssignedTo], [Microsoft.VSTS.Common.StackRank] FROM WorkItems"
	if len(where1) > 0 {
		wiql1 += " WHERE " + strings.Join(where1, " AND ")
	}
	wiql1 += " ORDER BY [Microsoft.VSTS.Common.StackRank] ASC, [System.ChangedDate] DESC"
	items1, err := queryItemsByWIQL(wiql1)
	if err != nil {
		return nil, err
	}

	// Second: all other types by ChangedDate DESC
	var where2 []string
	if !includeClosed {
		where2 = append(where2, "[System.State] <> 'Closed'")
	}
	where2 = append(where2, "[System.WorkItemType] NOT IN ('User Story','Bug')")
	wiql2 := "SELECT [System.Id], [System.Title], [System.State], [System.WorkItemType], [System.AssignedTo] FROM WorkItems"
	if len(where2) > 0 {
		wiql2 += " WHERE " + strings.Join(where2, " AND ")
	}
	wiql2 += " ORDER BY [System.ChangedDate] DESC"
	items2, err := queryItemsByWIQL(wiql2)
	if err != nil {
		return nil, err
	}

	return append(items1, items2...), nil
}

// queryItemsByWIQL runs a WIQL and parses into []queryItem supporting several shapes.
func queryItemsByWIQL(wiql string) ([]queryItem, error) {
	raw, err := az.QueryWIQL(wiql)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	var arr []queryItem
	if err := json.Unmarshal(raw, &arr); err == nil {
		// If it's an array shape, return it (even if empty)
		return arr, nil
	}
	var res struct {
		WorkItems []queryItem `json:"workItems"`
		Value     []queryItem `json:"value"`
	}
	if err := json.Unmarshal(raw, &res); err == nil {
		items := res.WorkItems
		if len(items) == 0 {
			items = res.Value
		}
		// Return items (possibly empty) to avoid errors on empty result sets
		return items, nil
	}
	// Fallback: no recognized shape; treat as empty rather than hard error
	return []queryItem{}, nil
}

// baseListWIQLWith builds WIQL using provided includeClosed and optional type filter.
func baseListWIQLWith(includeClosed bool, typeFilter string) string {
	wiql := "SELECT [System.Id], [System.Title], [System.State], [System.WorkItemType], [System.AssignedTo] FROM WorkItems"
	var where string
	if !includeClosed {
		where = "[System.State] <> 'Closed'"
	}
	if strings.TrimSpace(typeFilter) != "" {
		cond := fmt.Sprintf("[System.WorkItemType] = '%s'", typeFilter)
		if where == "" {
			where = cond
		} else {
			where = where + " AND " + cond
		}
	}
	if where != "" {
		wiql += " WHERE " + where
	}
	wiql += " ORDER BY [System.ChangedDate] DESC"
	return wiql
}

// queryItemsByParent lists direct children using System.Parent
func queryItemsByParent(parentID string, includeClosed bool) ([]queryItem, error) {
	// Step 1: fetch child IDs via WorkItems WIQL with System.Parent
	wiqlIDs := fmt.Sprintf("SELECT [System.Id] FROM WorkItems WHERE [System.Parent] = %s", parentID)
	raw, err := az.QueryWIQL(wiqlIDs)
	if err != nil {
		return nil, fmt.Errorf("query child ids failed: %w", err)
	}
	// Try to parse multiple possible shapes
	ids := make([]int, 0)
	// shape A: {"workItems":[{"id":123},...]}
	var resA struct {
		WorkItems []struct {
			ID     int            `json:"id"`
			Fields map[string]any `json:"fields"`
		} `json:"workItems"`
	}
	if err := json.Unmarshal(raw, &resA); err == nil && len(resA.WorkItems) > 0 {
		for _, it := range resA.WorkItems {
			if it.ID != 0 {
				ids = append(ids, it.ID)
				continue
			}
			if it.Fields != nil {
				if v, ok := it.Fields["System.Id"]; ok {
					switch n := v.(type) {
					case float64:
						ids = append(ids, int(n))
					}
				}
			}
		}
	}
	// shape B: array of objects with id/fields
	if len(ids) == 0 {
		var arr []map[string]any
		if err := json.Unmarshal(raw, &arr); err == nil && len(arr) > 0 {
			for _, obj := range arr {
				if v, ok := obj["id"]; ok {
					if f, ok := v.(float64); ok {
						ids = append(ids, int(f))
						continue
					}
				}
				if flds, ok := obj["fields"].(map[string]any); ok {
					if v, ok := flds["System.Id"]; ok {
						if f, ok := v.(float64); ok {
							ids = append(ids, int(f))
						}
					}
				}
			}
		}
	}
	if len(ids) == 0 {
		return []queryItem{}, nil
	}
	// Step 2: fetch fields for those IDs in one shot
	idStrs := make([]string, 0, len(ids))
	for _, id := range ids {
		idStrs = append(idStrs, strconv.Itoa(id))
	}
	wiql := fmt.Sprintf("SELECT [System.Id], [System.Title], [System.State], [System.WorkItemType], [System.AssignedTo] FROM WorkItems WHERE [System.Id] IN (%s)", strings.Join(idStrs, ","))
	if !includeClosed {
		wiql += " AND [System.State] <> \"Closed\""
	}
	wiql += " ORDER BY [System.ChangedDate] DESC"
	raw2, err := az.QueryWIQL(wiql)
	if err != nil {
		return nil, fmt.Errorf("query children failed: %w", err)
	}
	// Try array shape first
	var arr []queryItem
	if err := json.Unmarshal(raw2, &arr); err == nil && len(arr) > 0 {
		return arr, nil
	}
	var res struct {
		WorkItems []queryItem `json:"workItems"`
		Value     []queryItem `json:"value"`
	}
	if err := json.Unmarshal(raw2, &res); err != nil {
		return nil, fmt.Errorf("parse children: %w", err)
	}
	items := res.WorkItems
	if len(items) == 0 {
		items = res.Value
	}
	return items, nil
}

// baseListWIQL builds a WIQL string returning all needed fields for the filters.
func baseListWIQL(typeFilter string) string {
	wiql := "SELECT [System.Id], [System.Title], [System.State], [System.WorkItemType], [System.AssignedTo] FROM WorkItems"
	var where string
	if !includeAll {
		where = "[System.State] <> \"Closed\""
	}
	if typeFilter != "" {
		cond := fmt.Sprintf("[System.WorkItemType] = \"%s\"", typeFilter)
		if where == "" {
			where = cond
		} else {
			where = where + " AND " + cond
		}
	}
	if where != "" {
		wiql += " WHERE " + where
	}
	wiql += " ORDER BY [System.ChangedDate] DESC"
	return wiql
}

// renderList fetches details and prints a glow-style rendered markdown table to stdout.
func renderItems(items []queryItem) (string, error) {
	if len(items) == 0 {
		fmt.Println("No work-items found.")
		return "# Work Items\n\nNo work-items found.\n", nil
	}
	// Preserve order from WIQL; do not resort here
	// Build Markdown table
	var b bytes.Buffer
	b.WriteString("# Work Items\n\n")
	b.WriteString("| ID | Type | State | Assignee | Title |\n")
	b.WriteString("|---:|:-----|:------|:---------|:------|\n")
	// Resolve current user's displayName for bolding
	meDisplay, _ := az.CurrentUserDisplayName()
	for _, wi := range items {
		t := util.FieldString(wi.Fields, "System.WorkItemType")
		s := util.FieldString(wi.Fields, "System.State")
		ass := assigneeDisplay(wi.Fields)
		title := util.FieldString(wi.Fields, "System.Title")
		// Avoid breaking the table by escaping pipes
		title = strings.ReplaceAll(title, "|", "\\|")
		if s == "Active" && ass == meDisplay && meDisplay != "" {
			fmt.Fprintf(&b, "| **%d** | **%s** | **%s** | **%s** | **%s** |\n", wi.ID, t, s, ass, title)
		} else {
			fmt.Fprintf(&b, "| %d | %s | %s | %s | %s |\n", wi.ID, t, s, ass, title)
		}
	}
	md := b.String()
	// Render with glamour and wrap to terminal width
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(term.DetectWidth()),
		glamour.WithPreservedNewLines(),
	)
	if err != nil {
		return md, err
	}
	out, err := r.Render(md)
	if err != nil {
		return md, err
	}
	fmt.Print(out)
	return md, nil
}

// renderItemsTypeLess renders heading with columns without Type: ID | State | Assignee | Title
func renderItemsTypeLess(items []queryItem, heading string) (string, error) {
	if len(items) == 0 {
		fmt.Println("No work-items found.")
		return "# Work Items\n\nNo work-items found.\n", nil
	}
	// Preserve order from WIQL; do not resort here
	var b bytes.Buffer
	if heading == "" {
		heading = "Work Items"
	}
	b.WriteString("# " + heading + "\n\n")
	b.WriteString("| ID | State | Assignee | Title |\n")
	b.WriteString("|---:|:------|:---------|:------|\n")
	// Resolve current user's displayName for bolding
	meDisplay, _ := az.CurrentUserDisplayName()
	for _, wi := range items {
		s := util.FieldString(wi.Fields, "System.State")
		ass := assigneeDisplay(wi.Fields)
		title := util.FieldString(wi.Fields, "System.Title")
		title = strings.ReplaceAll(title, "|", "\\|")
		if s == "Active" && ass == meDisplay && meDisplay != "" {
			fmt.Fprintf(&b, "| **%d** | **%s** | **%s** | **%s** |\n", wi.ID, s, ass, title)
		} else {
			fmt.Fprintf(&b, "| %d | %s | %s | %s |\n", wi.ID, s, ass, title)
		}
	}
	md := b.String()
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(term.DetectWidth()),
		glamour.WithPreservedNewLines(),
	)
	if err != nil {
		return md, err
	}
	out, err := r.Render(md)
	if err != nil {
		return md, err
	}
	fmt.Print(out)
	return md, nil
}

// termWidth returns the terminal width using $COLUMNS or a sane default.
// width helpers moved to internal/term

// no duplicate fieldString; use util.FieldString instead

// renderChildren prints Parent info and a children table with assignee
func renderChildren(parent *az.WorkItem, items []queryItem) (string, error) {
	var b bytes.Buffer
	// Parent section
	b.WriteString("# Parent\n\n")
	b.WriteString("| ID | Column/State | Assignee | Title |\n")
	b.WriteString("|---:|:-------------|:---------|:------|\n")
	// Resolve current user's displayName for bolding
	meDisplay, _ := az.CurrentUserDisplayName()
	if parent != nil {
		state := util.FieldString(parent.Fields, "System.State")
		_, col := util.FindKanbanColumn(parent.Fields)
		colState := state
		if col != "" {
			colState = fmt.Sprintf("%s (%s)", col, state)
		}
		ass := assigneeDisplay(parent.Fields)
		title := util.FieldString(parent.Fields, "System.Title")
		title = strings.ReplaceAll(title, "|", "\\|")
		if state == "Active" && ass == meDisplay && meDisplay != "" {
			fmt.Fprintf(&b, "| **%d** | **%s** | **%s** | **%s** |\n\n", parent.ID, colState, ass, title)
		} else {
			fmt.Fprintf(&b, "| %d | %s | %s | %s |\n\n", parent.ID, colState, ass, title)
		}
	} else {
		b.WriteString("|(unknown)| | | |\n\n")
	}
	// Children table
	if len(items) == 0 {
		b.WriteString("No work-items found.\n")
	} else {
		sort.Slice(items, func(i, j int) bool { return items[i].ID > items[j].ID })
		b.WriteString("# Work Items\n\n")
		b.WriteString("| ID | Type | State | Assignee | Title |\n")
		b.WriteString("|---:|:-----|:------|:---------|:------|\n")
		for _, wi := range items {
			t := util.FieldString(wi.Fields, "System.WorkItemType")
			s := util.FieldString(wi.Fields, "System.State")
			title := util.FieldString(wi.Fields, "System.Title")
			ass := assigneeDisplay(wi.Fields)
			title = strings.ReplaceAll(title, "|", "\\|")
			if s == "Active" && ass == meDisplay && meDisplay != "" {
				fmt.Fprintf(&b, "| **%d** | **%s** | **%s** | **%s** | **%s** |\n", wi.ID, t, s, ass, title)
			} else {
				fmt.Fprintf(&b, "| %d | %s | %s | %s | %s |\n", wi.ID, t, s, ass, title)
			}
		}
	}
	md := b.String()
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(term.DetectWidth()),
		glamour.WithPreservedNewLines(),
	)
	if err != nil {
		return md, err
	}
	out, err := r.Render(md)
	if err != nil {
		return md, err
	}
	fmt.Print(out)
	return md, nil
}

// huhSavePath prompts for a file path using huh; path is prefilled and updated.
func huhSavePath(path *string) error {
	inp := huh.NewInput().Title("Save markdown as").Value(path)
	return huh.NewForm(huh.NewGroup(inp)).Run()
}

func assigneeDisplay(fields map[string]interface{}) string {
	s := util.FieldString(fields, "System.AssignedTo")
	if s != "" {
		return s
	}
	if v, ok := fields["System.AssignedTo"]; ok {
		if m, ok := v.(map[string]any); ok {
			if dn, ok := m["displayName"].(string); ok {
				return dn
			}
		}
	}
	return ""
}
