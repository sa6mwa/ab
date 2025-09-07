package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
)

// pickNonClosedID shows a huh picker of non-Closed items: "ID | T | Title"
// and returns the selected ID as a string.
func pickNonClosedID() (string, error) {
	items, err := queryItemsWithOrder("", false, poOrderGlobal)
	if err != nil {
		return "", err
	}
	if len(items) == 0 {
		return "", fmt.Errorf("no items to select")
	}
	type opt struct {
		id    int
		label string
	}
	opts := make([]opt, 0, len(items))
	for _, it := range items {
		title := utilField(it.Fields, "System.Title")
		if title == "" {
			title = "(no title)"
		}
		t := utilField(it.Fields, "System.WorkItemType")
		initial := ""
		if t != "" {
			initial = strings.ToUpper(t[:1])
		}
		opts = append(opts, opt{id: it.ID, label: fmt.Sprintf("%d | %s | %s", it.ID, initial, title)})
	}
	sel := huh.NewSelect[string]().Title("Pick work-item")
	var options []huh.Option[string]
	for _, o := range opts {
		options = append(options, huh.NewOption(o.label, strconv.Itoa(o.id)))
	}
	var chosen string
	sel = sel.Options(options...).Value(&chosen)
	if err := huh.NewForm(huh.NewGroup(sel)).Run(); err != nil {
		return "", err
	}
	if strings.TrimSpace(chosen) == "" {
		return "", fmt.Errorf("no selection")
	}
	return chosen, nil
}

// pickNonClosedIDs shows a multi-select picker and returns selected IDs.
func pickNonClosedIDs() ([]string, error) {
	items, err := queryItemsWithOrder("", false, poOrderGlobal)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("no items to select")
	}
	type opt struct {
		id    int
		label string
	}
	opts := make([]opt, 0, len(items))
	for _, it := range items {
		title := utilField(it.Fields, "System.Title")
		if title == "" {
			title = "(no title)"
		}
		t := utilField(it.Fields, "System.WorkItemType")
		initial := ""
		if t != "" {
			initial = strings.ToUpper(t[:1])
		}
		opts = append(opts, opt{id: it.ID, label: fmt.Sprintf("%d | %s | %s", it.ID, initial, title)})
	}
	// Build options
	var options []huh.Option[string]
	for _, o := range opts {
		options = append(options, huh.NewOption(o.label, strconv.Itoa(o.id)))
	}
	var chosen []string
	msel := huh.NewMultiSelect[string]().Title("Pick work-items").Options(options...).Value(&chosen)
	if err := huh.NewForm(huh.NewGroup(msel)).Run(); err != nil {
		return nil, err
	}
	if len(chosen) == 0 {
		return nil, fmt.Errorf("no selection")
	}
	return chosen, nil
}
