package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/areyesgfx/swale/internal/generate"
	"github.com/areyesgfx/swale/internal/spec"
)

// initAnswers is what both "swale init" front doors collect: flags (for
// scripts and CI) and the interactive wizard. Each front door only fills
// this in; toSpec and spec.Save own the rules, so the two can't drift.
// accountName is the only account in single-account mode and the
// management account in multi-account mode.
type initAnswers struct {
	region       string
	multiAccount bool
	accountName  string
	envs         []envAnswers
}

type envAnswers struct {
	name    string
	vpcCIDR string
	nat     spec.NATMode
}

// toSpec builds the Spec without validating it; spec.Save does that.
// Multi-account mode names each workload account after its environment.
func (a initAnswers) toSpec() spec.Spec {
	s := spec.Spec{Region: a.region}
	s.Accounts.Mode = spec.AccountModeSingle
	if a.multiAccount {
		s.Accounts.Mode = spec.AccountModeMulti
	}
	s.Accounts.Accounts = []spec.Account{{Name: a.accountName, IsManagement: a.multiAccount}}

	for _, e := range a.envs {
		account := a.accountName
		if a.multiAccount {
			account = e.name
			s.Accounts.Accounts = append(s.Accounts.Accounts, spec.Account{Name: e.name})
		}
		nat := e.nat
		if nat == "" {
			nat = spec.DefaultNAT(e.name)
		}
		s.Environments = append(s.Environments, spec.Environment{
			Name:       e.name,
			Account:    account,
			Networking: spec.NetworkingSpec{VPCCIDR: e.vpcCIDR, NAT: nat},
		})
	}
	return s
}

func newInitCmd() *cobra.Command {
	var f initFlags

	cmd := &cobra.Command{
		Use:   "init [dir]",
		Short: "Create a new swale project",
		Long: `Create a new swale project in dir (default: the current directory): a
swale.yaml spec, and the Terragrunt/Terraform project generated from it.

Run with no flags to fill in an interactive form. Pass flags instead to run
non-interactively, e.g. in scripts or CI.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := projectDir(args)
			specPath := filepath.Join(dir, spec.FileName)
			// Checked before asking anything, so nobody fills in the whole
			// form only to be told the project already exists.
			if _, err := os.Stat(specPath); err == nil {
				return fmt.Errorf("%s already exists; edit it and run \"swale generate\", or delete it to start over", specPath)
			} else if !errors.Is(err, fs.ErrNotExist) {
				return err
			}

			var (
				a   initAnswers
				err error
			)
			if cmd.Flags().NFlag() == 0 {
				a, err = runWizard()
			} else {
				a, err = f.answers()
			}
			if err != nil {
				return err
			}

			// Render before writing anything, so a spec swale can't generate
			// leaves no half-created project behind.
			s := a.toSpec()
			if err := s.Validate(); err != nil {
				return err
			}
			files, err := generate.Render(s)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
			if err := spec.Save(s, specPath); err != nil {
				return err
			}
			r, err := generate.Write(dir, files)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Created %s and generated your project:\n", specPath)
			printResult(out, r)
			printNextSteps(out, dir, s)
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&f.region, "region", "", "AWS region generated infrastructure is deployed into, e.g. us-east-1")
	flags.StringArrayVar(&f.envs, "env", nil, "environment name (repeatable), e.g. --env dev --env prod")
	flags.BoolVar(&f.singleAccount, "single-account", false, "use a single AWS account shared by every environment")
	flags.BoolVar(&f.multiAccount, "multi-account", false, "use one AWS account per environment plus a management account")
	flags.StringVar(&f.accountName, "account-name", "main", "account name to use in --single-account mode")
	flags.StringVar(&f.managementAccount, "management-account", "", "management account name, required in --multi-account mode")
	flags.StringArrayVar(&f.vpcCIDRs, "vpc-cidr", nil, "VPC CIDR for an environment, as env=cidr (repeatable, required for every --env)")
	flags.StringArrayVar(&f.natModes, "nat", nil, `NAT mode for an environment, as env=single|per_az (repeatable, optional; defaults to per_az for "prod", single otherwise)`)

	cmd.MarkFlagsMutuallyExclusive("single-account", "multi-account")
	cmd.MarkFlagsMutuallyExclusive("single-account", "management-account")
	cmd.MarkFlagsMutuallyExclusive("multi-account", "account-name")

	return cmd
}

type initFlags struct {
	region            string
	envs              []string
	singleAccount     bool
	multiAccount      bool
	accountName       string
	managementAccount string
	vpcCIDRs          []string
	natModes          []string
}

func (f initFlags) answers() (initAnswers, error) {
	if !f.singleAccount && !f.multiAccount {
		return initAnswers{}, errors.New("specify one of --single-account or --multi-account")
	}
	accountName := f.accountName
	if f.multiAccount {
		if f.managementAccount == "" {
			return initAnswers{}, errors.New("--management-account is required with --multi-account")
		}
		accountName = f.managementAccount
	}

	known := make(map[string]bool, len(f.envs))
	for _, e := range f.envs {
		known[e] = true
	}
	cidrs, err := parsePairs("vpc-cidr", f.vpcCIDRs, known)
	if err != nil {
		return initAnswers{}, err
	}
	nats, err := parsePairs("nat", f.natModes, known)
	if err != nil {
		return initAnswers{}, err
	}

	a := initAnswers{region: f.region, multiAccount: f.multiAccount, accountName: accountName}
	for _, e := range f.envs {
		a.envs = append(a.envs, envAnswers{name: e, vpcCIDR: cidrs[e], nat: spec.NATMode(nats[e])})
	}
	return a, nil
}

// parsePairs parses repeated "env=value" flag values, rejecting malformed
// pairs, envs not passed via --env, and duplicates.
func parsePairs(flag string, pairs []string, envs map[string]bool) (map[string]string, error) {
	result := make(map[string]string, len(pairs))
	for _, p := range pairs {
		env, value, ok := strings.Cut(p, "=")
		if !ok || env == "" || value == "" {
			return nil, fmt.Errorf("--%s: %q must be in the form env=value", flag, p)
		}
		if !envs[env] {
			return nil, fmt.Errorf("--%s: env %q was not passed via --env", flag, env)
		}
		if _, dup := result[env]; dup {
			return nil, fmt.Errorf("--%s: env %q specified more than once", flag, env)
		}
		result[env] = value
	}
	return result, nil
}
