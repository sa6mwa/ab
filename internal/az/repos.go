package az

import (
	"encoding/json"
	"fmt"
)

// Repo represents a minimal Azure DevOps repository shape from `az repos list`.
type Repo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Size      int64  `json:"size,omitempty"`
	SSHURL    string `json:"sshUrl"`
	RemoteURL string `json:"remoteUrl"`
	WebURL    string `json:"webUrl"`
}

// ListRepos returns repositories for the current az devops defaults.
func ListRepos() ([]Repo, error) {
	out, err := runAz("repos", "list", "-o", "json")
	if err != nil {
		return nil, err
	}
	var repos []Repo
	if err := json.Unmarshal(out, &repos); err != nil {
		return nil, err
	}
	return repos, nil
}

// CreateRepo creates a repository and returns its JSON info.
func CreateRepo(name string) (*Repo, error) {
	out, err := runAz("repos", "create", "--name", name, "-o", "json")
	if err != nil {
		return nil, err
	}
	var r Repo
	if err := json.Unmarshal(out, &r); err != nil {
		return nil, err
	}
	if r.Name == "" {
		return nil, fmt.Errorf("unexpected create output")
	}
	return &r, nil
}

// DeleteRepo deletes a repository by name or id. Pass --yes to az to avoid its prompt.
func DeleteRepo(nameOrID string, assumeYes bool) error {
	// Prefer deleting by name to avoid ambiguity; az accepts --repository for name.
	args := []string{"repos", "delete", "--repository", nameOrID}
	if assumeYes {
		args = append(args, "--yes")
	}
	_, err := runAz(args...)
	return err
}
