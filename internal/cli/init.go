package cli

import (
	"github.com/spf13/cobra"

	"github.com/areyesgfx/swale/internal/spec"
)

// newInitCmd builds the "swale init" subcommand: the direct-CLI-flags front
// door that turns flags into a Spec and writes it out as swale.yaml. The
// wizard (huh-based) front door coming in Phase 2 must produce the
// identical Spec shape via the same buildSpec/spec.Save path.
func newInitCmd() *cobra.Command {
	var (
		opts   initOptions
		output string
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a new swale.yaml spec",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.accountNameSet = cmd.Flags().Changed("account-name")
			opts.managementAccountSet = cmd.Flags().Changed("management-account")

			s, err := buildSpec(opts)
			if err != nil {
				return err
			}
			if err := spec.Save(s, output); err != nil {
				return err
			}
			cmd.Printf("wrote %s\n", output)
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringArrayVar(&opts.envs, "env", nil, "environment name (repeatable), e.g. --env dev --env prod")
	flags.BoolVar(&opts.singleAccount, "single-account", false, "use a single AWS account shared by every environment")
	flags.BoolVar(&opts.multiAccount, "multi-account", false, "use one AWS account per environment plus a management account")
	flags.StringVar(&opts.accountName, "account-name", "main", "account name to use in --single-account mode")
	flags.StringVar(&opts.managementAccount, "management-account", "", "management account name, required in --multi-account mode")
	flags.StringArrayVar(&opts.vpcCIDRs, "vpc-cidr", nil, "VPC CIDR for an environment, as env=cidr (repeatable, required for every --env)")
	flags.StringArrayVar(&opts.natModes, "nat", nil, `NAT mode for an environment, as env=single|per_az (repeatable, optional; defaults to per_az for "prod", single otherwise)`)
	flags.StringVarP(&output, "output", "o", "swale.yaml", "path to write the spec")

	return cmd
}
