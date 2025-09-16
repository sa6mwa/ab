package az

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	shellescape "al.essio.dev/pkg/shellescape"
	"github.com/charmbracelet/huh"
)

var ErrCancelled = errors.New("cancelled")

// azExec is the executor used by runAz. Overridable in tests.
var azExec = realAzExec

// global silent flag for printing commands
var silent bool

// SetSilent controls whether to print the az command lines.
func SetSilent(s bool) { silent = s }

// SetExecutorForTest overrides the az executor. Intended for tests.
func SetExecutorForTest(exec func(args ...string) ([]byte, error)) {
	azExec = exec
}

// Confirmation modes
type ConfirmMode int

const (
	ConfirmAlways ConfirmMode = iota
	ConfirmMutations
	ConfirmNever
)

var confirmMode = ConfirmMutations

func init() {
	if v := strings.TrimSpace(os.Getenv("AB_CONFIRM")); v != "" {
		_ = SetConfirmMode(v)
	}
}

// SetConfirmMode sets the current confirmation policy (always|mutations|never).
func SetConfirmMode(mode string) error {
	m, err := parseConfirmMode(mode)
	if err != nil {
		return err
	}
	confirmMode = m
	return nil
}

func parseConfirmMode(mode string) (ConfirmMode, error) {
	v := strings.ToLower(strings.TrimSpace(mode))
	switch v {
	case "always", "all", "true", "on", "1", "yes", "y":
		return ConfirmAlways, nil
	case "mutations", "mutation", "writes", "write", "updates", "changes":
		return ConfirmMutations, nil
	case "never", "none", "false", "off", "0", "no", "n":
		return ConfirmNever, nil
	case "":
		return confirmMode, nil
	default:
		return confirmMode, fmt.Errorf("invalid confirm mode: %q (valid: always|mutations|never)", mode)
	}
}

// runAz prints and confirms the az command before execution, then returns stdout or error.
func runAz(args ...string) ([]byte, error) {
	// Print a safe-to-shell-copy command line using shellescape
	cmdline := shellescape.QuoteCommand(append([]string{"az"}, args...))
	if !silent {
		fmt.Fprintln(os.Stderr, cmdline)
	}
	if shouldConfirm(args) {
		var proceed bool
		confirm := huh.NewConfirm().Title("Run this command?").Description(cmdline).Affirmative("Yes").Negative("No").Value(&proceed)
		form := huh.NewForm(huh.NewGroup(confirm))
		if err := form.Run(); err != nil {
			return nil, err
		}
		if !proceed {
			return nil, ErrCancelled
		}
	}
	return azExec(args...)
}

// realAzExec executes the az command and returns stdout or error with stderr context.
func realAzExec(args ...string) ([]byte, error) {
	cmd := exec.Command("az", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("az %s failed: %v: %s", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.Bytes(), nil
}

func shouldConfirm(args []string) bool {
	switch confirmMode {
	case ConfirmNever:
		return false
	case ConfirmAlways:
		return true
	case ConfirmMutations:
		if len(args) == 0 {
			return false
		}
		// az rest --method <m>
		if args[0] == "rest" {
			for i := 1; i < len(args); i++ {
				if args[i] == "--method" && i+1 < len(args) {
					m := strings.ToLower(args[i+1])
					return m != "get"
				}
			}
			// No explicit method: be conservative
			return true
		}
		// az boards work-item <create|update|delete>
		for i := 0; i+2 < len(args); i++ {
			if args[i] == "boards" && args[i+1] == "work-item" {
				a := strings.ToLower(args[i+2])
				if a == "create" || a == "update" || a == "delete" {
					return true
				}
				if a == "relation" {
					// relation add/remove/delete are mutations
					if i+3 < len(args) {
						op := strings.ToLower(args[i+3])
						return op == "add" || op == "remove" || op == "delete"
					}
				}
				return false
			}
		}
		// Other commands are treated as reads by default
		return false
	default:
		return true
	}
}

func formatAz(args []string) string { return shellescape.QuoteCommand(append([]string{"az"}, args...)) }

// CurrentUserUPN returns the signed-in user's principal name (email) via az ad.
func CurrentUserUPN() (string, error) {
	out, err := runAz("ad", "signed-in-user", "show", "--query", "userPrincipalName", "-o", "tsv")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// CurrentUserDisplayName returns the signed-in user's display name via az ad.
func CurrentUserDisplayName() (string, error) {
	out, err := runAz("ad", "signed-in-user", "show", "--query", "displayName", "-o", "tsv")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// QueryWIQL returns the raw JSON output from az boards query for the provided WIQL string.
func QueryWIQL(wiql string) ([]byte, error) {
	return runAz("boards", "query", "--wiql", wiql, "-o", "json")
}

// WorkItem is a minimal shape for az boards work-item show output.
type WorkItem struct {
	ID     int                    `json:"id"`
	Rev    int                    `json:"rev"`
	Fields map[string]interface{} `json:"fields"`
	URL    string                 `json:"url"`
}

// ShowWorkItem gets a work item as JSON bytes and optionally decodes it.
func ShowWorkItem(id string) ([]byte, *WorkItem, error) {
	raw, err := runAz("boards", "work-item", "show", "--id", id, "-o", "json")
	if err != nil {
		return nil, nil, err
	}
	var wi WorkItem
	if err := json.Unmarshal(raw, &wi); err != nil {
		// still return raw for passthrough
		return raw, nil, nil
	}
	return raw, &wi, nil
}

// UpdateWorkItemFields updates fields on a work item and returns raw JSON output.
func UpdateWorkItemFields(id string, fields map[string]string) ([]byte, error) {
	args := []string{"boards", "work-item", "update", "--id", id}
	// Build --fields list as repeated args: --fields "A=B" "C=D"
	if len(fields) > 0 {
		args = append(args, "--fields")
		for k, v := range fields {
			args = append(args, fmt.Sprintf("%s=%s", k, v))
		}
	}
	args = append(args, "-o", "json")
	return runAz(args...)
}

// CreateWorkItem creates a new work item of a specific type with fields and optional relation.
func CreateWorkItem(wiType, title string, fields map[string]string, relation string) ([]byte, error) {
	args := []string{"boards", "work-item", "create", "--type", wiType, "--title", title}
	if len(fields) > 0 {
		args = append(args, "--fields")
		for k, v := range fields {
			args = append(args, fmt.Sprintf("%s=%s", k, v))
		}
	}
	args = append(args, "-o", "json")
	return runAz(args...)
}

// AddWorkItemRelation adds a relation from a work item to a target work item.
func AddWorkItemRelation(id, relationType, targetID string) ([]byte, error) {
	args := []string{"boards", "work-item", "relation", "add", "--id", id, "--relation-type", relationType, "--target-id", targetID, "-o", "json"}
	return runAz(args...)
}

// DeleteWorkItem deletes a work item by ID.
func DeleteWorkItem(id string) ([]byte, error) {
	args := []string{"boards", "work-item", "delete", "--id", id, "--yes", "-o", "json"}
	return runAz(args...)
}

// UpdateWorkItemAssignee updates the assigned-to field using the dedicated flag.
func UpdateWorkItemAssignee(id, assignee string) ([]byte, error) {
	args := []string{"boards", "work-item", "update", "--id", id, "--assigned-to", assignee, "-o", "json"}
	return runAz(args...)
}

// PrintJSON writes raw JSON bytes to stdout without modification.
func PrintJSON(raw []byte) error {
	_, err := os.Stdout.Write(raw)
	if err == nil && len(raw) > 0 && raw[len(raw)-1] != '\n' {
		// ensure newline for cleanliness
		_, _ = os.Stdout.Write([]byte("\n"))
	}
	return err
}

// DevOpsDefaults holds current az devops defaults (org, project) and default team.
type DevOpsDefaults struct {
	Organization string
	Project      string
	Team         string
}

// GetDevOpsDefaults retrieves the configured default organization and project, and resolves the project's default team.
func GetDevOpsDefaults() (*DevOpsDefaults, error) {
	out, err := runAz("devops", "configure", "-l", "-o", "json")
	if err != nil {
		return nil, err
	}
	var cfg struct {
		Defaults map[string]string `json:"defaults"`
	}
	_ = json.Unmarshal(out, &cfg) // best-effort
	org := cfg.Defaults["organization"]
	proj := cfg.Defaults["project"]
	if proj == "" {
		return nil, fmt.Errorf("az devops default project not set; run 'az devops configure --defaults project=<name> organization=<url>'")
	}
	// Resolve default team name
	pjson, err := runAz("devops", "project", "show", "--project", proj, "-o", "json")
	if err != nil {
		return nil, err
	}
	var p struct {
		DefaultTeam struct {
			Name string `json:"name"`
		} `json:"defaultTeam"`
	}
	if err := json.Unmarshal(pjson, &p); err != nil || p.DefaultTeam.Name == "" {
		return nil, fmt.Errorf("unable to resolve default team for project %q", proj)
	}
	return &DevOpsDefaults{Organization: org, Project: proj, Team: p.DefaultTeam.Name}, nil
}

// azRestGET performs an authenticated GET using az rest and returns raw json bytes.
func azRestGET(url string) ([]byte, error) { return runAz("rest", "--method", "get", "--url", url) }

// Board and Column shapes for Azure Boards REST
type Board struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type BoardsList struct {
	Value []Board `json:"value"`
}
type BoardColumn struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	IsSplit      bool              `json:"isSplit"`
	ColumnType   string            `json:"columnType"`
	StateMapping map[string]string `json:"stateMappings"`
}
type ColumnsList struct {
	Value []BoardColumn `json:"value"`
}

// BoardColumnsForType returns ordered column names and split flags for the default team's board that supports the given work item type.
func BoardColumnsForType(wiType string) (columns []BoardColumn, err error) {
	defs, err := GetDevOpsDefaults()
	if err != nil {
		return nil, err
	}
	base := strings.TrimRight(defs.Organization, "/")
	// List boards for team
	blRaw, err := azRestGET(fmt.Sprintf("%s/%s/%s/_apis/work/boards?api-version=7.0", base, defs.Project, defs.Team))
	if err != nil {
		return nil, err
	}
	var bl BoardsList
	if err := json.Unmarshal(blRaw, &bl); err != nil {
		return nil, err
	}
	// Find a board whose columns have stateMappings for our type
	for _, b := range bl.Value {
		colsRaw, err := azRestGET(fmt.Sprintf("%s/%s/%s/_apis/work/boards/%s/columns?api-version=7.0", base, defs.Project, defs.Team, b.ID))
		if err != nil {
			continue
		}
		var cl ColumnsList
		if err := json.Unmarshal(colsRaw, &cl); err != nil {
			continue
		}
		for _, c := range cl.Value {
			if _, ok := c.StateMapping[wiType]; ok {
				return cl.Value, nil
			}
		}
	}
	return nil, fmt.Errorf("no board columns found for type %q; ensure the default team board includes this type", wiType)
}
