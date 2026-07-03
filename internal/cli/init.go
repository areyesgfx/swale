package cli

import "github.com/spf13/cobra"

// newInitCmd builds the "swale init" subcommand. It has no behavior yet;
// SWA-10 wires it up to produce a swale.yaml spec from CLI flags.
func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create a new swale.yaml spec",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Println("swale init: not yet implemented")
			return nil
		},
	}
}
