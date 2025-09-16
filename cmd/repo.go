package cmd

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/sa6mwa/ab/internal/az"
	"github.com/sa6mwa/ab/internal/git"
	"github.com/sa6mwa/ab/internal/openurl"
	"github.com/spf13/cobra"
)

// Shared flags
var repoHTTPS bool

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Work with Azure Repos (gh-style)",
	Long:  "Manage Azure Repos: pick, view, clone, create, list, delete.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default: show repo picker, then action picker
		repos, err := az.ListRepos()
		if err != nil {
			return err
		}
		if len(repos) == 0 {
			return fmt.Errorf("no repositories found; check az devops defaults")
		}
		sort.Slice(repos, func(i, j int) bool { return strings.ToLower(repos[i].Name) < strings.ToLower(repos[j].Name) })
		var options []huh.Option[int]
		for i, r := range repos {
			label := fmt.Sprintf("%s | %s | %s", r.Name, r.ID, humanSize(r.Size))
			options = append(options, huh.NewOption(label, i))
		}
		var chosen int
		sel := huh.NewSelect[int]().Title("Select repository").Options(options...).Value(&chosen)
		if err := huh.NewForm(huh.NewGroup(sel)).Run(); err != nil {
			return err
		}
		r := repos[chosen]
		// Action picker (Clone SSH, Clone HTTP, View, Cancel)
		ssh := strings.TrimSpace(r.SSHURL)
		http := strings.TrimSpace(r.RemoteURL)
		web := strings.TrimSpace(r.WebURL)
		type action int
		const (
			actCloneSSH action = iota
			actCloneHTTP
			actView
			actCancel
		)
		var actOptions []huh.Option[action]
		actOptions = append(actOptions, huh.NewOption[action](fmt.Sprintf("Clone %s (ssh)", ssh), actCloneSSH))
		actOptions = append(actOptions, huh.NewOption[action](fmt.Sprintf("Clone %s (http)", http), actCloneHTTP))
		actOptions = append(actOptions, huh.NewOption[action](fmt.Sprintf("View %s", web), actView))
		actOptions = append(actOptions, huh.NewOption[action]("Cancel", actCancel))
		var actChosen action
		actSel := huh.NewSelect[action]().Title("Action").Options(actOptions...).Value(&actChosen)
		if err := huh.NewForm(huh.NewGroup(actSel)).Run(); err != nil {
			return err
		}
		switch actChosen {
		case actCloneSSH:
			return git.Clone(ssh, silentFlag)
		case actCloneHTTP:
			return git.Clone(http, silentFlag)
		case actView:
			return openurl.Open(web)
		case actCancel:
			return az.ErrCancelled
		default:
			return nil
		}
	},
}

// repo view/show
var repoViewCmd = &cobra.Command{
	Use:     "view [repository]",
	Aliases: []string{"show"},
	Short:   "Open the repository in the browser",
	Args:    cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var r *az.Repo
		var err error
		if len(args) == 0 {
			r, err = pickRepo()
			if err != nil {
				return err
			}
		} else {
			r, err = findRepo(args[0])
			if err != nil {
				return err
			}
		}
		return openurl.Open(strings.TrimSpace(r.WebURL))
	},
}

// repo clone
var repoCloneCmd = &cobra.Command{
	Use:   "clone [repository]",
	Short: "Clone a repository (SSH by default)",
	Args:  cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			r, err := findRepo(args[0])
			if err != nil {
				return err
			}
			if repoHTTPS {
				return git.Clone(strings.TrimSpace(r.RemoteURL), silentFlag)
			}
			return git.Clone(strings.TrimSpace(r.SSHURL), silentFlag)
		}
		// Picker: select then clone
		r, err := pickRepo()
		if err != nil {
			return err
		}
		if repoHTTPS {
			return git.Clone(strings.TrimSpace(r.RemoteURL), silentFlag)
		}
		return git.Clone(strings.TrimSpace(r.SSHURL), silentFlag)
	},
}

// repo list
var repoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List repositories",
	RunE: func(cmd *cobra.Command, args []string) error {
		repos, err := az.ListRepos()
		if err != nil {
			return err
		}
		if len(repos) == 0 {
			return nil
		}
		sort.Slice(repos, func(i, j int) bool { return strings.ToLower(repos[i].Name) < strings.ToLower(repos[j].Name) })
		for _, r := range repos {
			fmt.Fprintf(os.Stdout, "%s | %s | %s\n", r.Name, r.ID, humanSize(r.Size))
		}
		return nil
	},
}

// repo create
var repoCreateClone bool
var repoCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a repository",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := strings.TrimSpace(args[0])
		if name == "" {
			return errors.New("repository name required")
		}
		r, err := az.CreateRepo(name)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "Created repo %s (%s)\n", r.Name, r.ID)
		if repoCreateClone {
			if repoHTTPS {
				return git.Clone(strings.TrimSpace(r.RemoteURL), silentFlag)
			}
			return git.Clone(strings.TrimSpace(r.SSHURL), silentFlag)
		}
		return nil
	},
}

// repo delete
var repoDeleteCmd = &cobra.Command{
	Use:   "delete <repository>",
	Short: "Delete a repository",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := strings.TrimSpace(args[0])
		// Always require confirmation unless global --yes was passed
		if !yesFlag {
			var proceed bool
			msg := fmt.Sprintf("This will permanently delete repo %q. Continue?", name)
			cf := huh.NewConfirm().Title("Confirm delete").Description(msg).Affirmative("Delete").Negative("Cancel").Value(&proceed)
			if err := huh.NewForm(huh.NewGroup(cf)).Run(); err != nil {
				return err
			}
			if !proceed {
				return az.ErrCancelled
			}
		}
		// Prevent az interactive prompt; we already confirmed above (or user passed --yes)
		return az.DeleteRepo(name, true)
	},
}

func init() {
	// Shared flags for protocol
	repoCmd.PersistentFlags().BoolVar(&repoHTTPS, "https", false, "Use https remoteUrl instead of sshUrl (alias: --http)")
	repoCmd.PersistentFlags().BoolVar(&repoHTTPS, "http", false, "Alias for --https")

	// Wire subcommands
	repoCmd.AddCommand(repoViewCmd)
	repoCmd.AddCommand(repoCloneCmd)
	repoCmd.AddCommand(repoListCmd)
	repoCmd.AddCommand(repoCreateCmd)
	repoCmd.AddCommand(repoDeleteCmd)

	// Create flags
	repoCreateCmd.Flags().BoolVarP(&repoCreateClone, "clone", "c", false, "Clone the repo after creation")

	rootCmd.AddCommand(repoCmd)
}

// Helpers
func pickRepo() (*az.Repo, error) {
	repos, err := az.ListRepos()
	if err != nil {
		return nil, err
	}
	if len(repos) == 0 {
		return nil, fmt.Errorf("no repositories found; check az devops defaults")
	}
	sort.Slice(repos, func(i, j int) bool { return strings.ToLower(repos[i].Name) < strings.ToLower(repos[j].Name) })
	var options []huh.Option[int]
	for i, r := range repos {
		options = append(options, huh.NewOption(fmt.Sprintf("%s | %s | %s", r.Name, r.ID, humanSize(r.Size)), i))
	}
	var chosen int
	sel := huh.NewSelect[int]().Title("Select repository").Options(options...).Value(&chosen)
	if err := huh.NewForm(huh.NewGroup(sel)).Run(); err != nil {
		return nil, err
	}
	r := repos[chosen]
	return &r, nil
}

func findRepo(nameOrID string) (*az.Repo, error) {
	nameOrID = strings.TrimSpace(nameOrID)
	repos, err := az.ListRepos()
	if err != nil {
		return nil, err
	}
	for _, r := range repos {
		if strings.EqualFold(r.Name, nameOrID) || r.ID == nameOrID {
			rr := r
			return &rr, nil
		}
	}
	return nil, fmt.Errorf("repository not found: %s", nameOrID)
}

func humanSize(n int64) string {
	if n <= 0 {
		return "0 B"
	}
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	exp := 0
	for n >= unit && exp < 4 {
		n /= unit
		exp++
	}
	suffix := []string{"KB", "MB", "GB", "TB"}[exp-1]
	return fmt.Sprintf("%d %s", n, suffix)
}
