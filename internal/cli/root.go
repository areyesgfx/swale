package cli

import "github.com/spf13/cobra"

// NewRootCmd builds the swale root command and wires up its subcommands.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "swale",
		Short: "Generate readable, editable Terraform/Terragrunt for AWS",
		Long: `swale is a self-service IaC platform: a CLI that generates readable,
editable Terraform/Terragrunt projects from a single spec (swale.yaml), so
developers without deep cloud knowledge can stand up secure AWS
infrastructure without hand-writing HCL from scratch.`,
		SilenceUsage: true,
	}

	root.AddCommand(newInitCmd())

	return root
}
