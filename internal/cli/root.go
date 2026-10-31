package cli

import (
	"crypto/rand"

	"github.com/spf13/cobra"
)

// NewRootCmd builds the swale root command and wires up its subcommands.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "swale",
		Short: "Set up secure AWS infrastructure for your app by answering a few questions",
		Long: `swale sets up AWS infrastructure for your app. You answer a few questions
about the app, and swale generates a readable, editable Terragrunt and
Terraform project for it, with secure defaults, that you deploy and own.

Start with "swale init".`,
		SilenceUsage: true,
	}

	root.AddCommand(newInitCmd(rand.Reader), newGenerateCmd())

	return root
}
