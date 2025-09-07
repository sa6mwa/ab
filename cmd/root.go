package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/sa6mwa/ab/internal/az"
	"github.com/sa6mwa/ab/internal/board"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "ab",
	Short:         "Azure Boards task helper",
	Long:          "ab is a thin wrapper around Azure CLI (az) boards commands for common workflows.",
	SilenceUsage:  true,
	SilenceErrors: true,
	Annotations: map[string]string{
		"author": "Michel Blomgren <michel.blomgren@nionit.com>",
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if yesFlag {
			if err := az.SetConfirmMode("never"); err != nil {
				return err
			}
		} else if confirmFlag != "" {
			if err := az.SetConfirmMode(confirmFlag); err != nil {
				return err
			}
		}
		az.SetSilent(silentFlag)
		if defaultColumnsFlag {
			// Explicit flag overrides any AB_COLUMNS env setting
			board.SetDefaultAgileColumns()
		}
		return nil
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var confirmFlag string
var yesFlag bool
var silentFlag bool
var defaultColumnsFlag bool

// Global PO order toggle, affects pickers and listings where applicable
var poOrderGlobal bool

func init() {
	// Version flag enables `--version`; default to dev unless overridden via -ldflags
	rootCmd.Version = "dev"

	// Include author in help and version output
	helpTmpl := rootCmd.HelpTemplate() + "\nAuthor: {{index .Annotations \"author\"}}\n"
	rootCmd.SetHelpTemplate(helpTmpl)
	rootCmd.SetVersionTemplate("{{.Name}} {{.Version}}\nAuthor: {{index .Annotations \"author\"}}\n")

	rootCmd.PersistentFlags().StringVar(&confirmFlag, "confirm", "", "Confirmation mode: always|mutations|never (overrides AB_CONFIRM)")
	rootCmd.PersistentFlags().BoolVarP(&yesFlag, "yes", "y", false, "Do not prompt; equivalent to --confirm never")
	rootCmd.PersistentFlags().BoolVarP(&silentFlag, "silent", "s", false, "Silent mode: do not print az commands, only outputs")
	rootCmd.PersistentFlags().BoolVarP(&defaultColumnsFlag, "default-columns", "d", false, "Use default Agile columns: New,Active,Resolved,Closed (overrides AB_COLUMNS)")
	rootCmd.PersistentFlags().BoolVarP(&poOrderGlobal, "po-order", "P", envTrue("AB_PO_ORDER") || envTrue("AB_STACKRANK"), "Order by PO priority where possible (StackRank for Stories/Bugs). Can be set via AB_PO_ORDER=true or AB_STACKRANK=true; flag overrides if provided")
}

func envTrue(name string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	switch v {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}
