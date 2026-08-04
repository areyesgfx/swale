package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/areyesgfx/swale/internal/spec"
)

// initOptions holds the raw "swale init" flag values, decoupled from Cobra
// so buildSpec can be tested without going through command execution. The
// *Set fields record whether the user explicitly passed a flag (vs. it
// carrying its zero/default value), which matters for flags that are only
// meaningful in one account mode.
type initOptions struct {
	region               string
	envs                 []string
	singleAccount        bool
	multiAccount         bool
	accountName          string
	accountNameSet       bool
	managementAccount    string
	managementAccountSet bool
	vpcCIDRs             []string // "env=cidr"
	natModes             []string // "env=single|per_az"
}

// buildSpec turns parsed "swale init" flags into a spec.Spec. It does not
// call spec.Validate itself: Save does that, and duplicating the checks
// here would just give two places for the rules to drift apart.
func buildSpec(opts initOptions) (spec.Spec, error) {
	if opts.region == "" {
		return spec.Spec{}, errors.New("--region is required")
	}
	if opts.singleAccount == opts.multiAccount {
		return spec.Spec{}, errors.New("specify exactly one of --single-account or --multi-account")
	}
	if len(opts.envs) == 0 {
		return spec.Spec{}, errors.New("--env is required (at least one environment)")
	}

	envSeen := make(map[string]struct{}, len(opts.envs))
	for _, e := range opts.envs {
		if e == "" {
			return spec.Spec{}, errors.New("--env: environment name must not be empty")
		}
		if _, dup := envSeen[e]; dup {
			return spec.Spec{}, fmt.Errorf("--env: %q specified more than once", e)
		}
		envSeen[e] = struct{}{}
	}

	vpcCIDRs, err := parsePerEnvFlag("vpc-cidr", opts.vpcCIDRs, envSeen)
	if err != nil {
		return spec.Spec{}, err
	}
	natModes, err := parsePerEnvFlag("nat", opts.natModes, envSeen)
	if err != nil {
		return spec.Spec{}, err
	}

	accounts, accountForEnv, err := buildAccounts(opts)
	if err != nil {
		return spec.Spec{}, err
	}

	envs := make([]spec.Environment, 0, len(opts.envs))
	for _, e := range opts.envs {
		cidr, ok := vpcCIDRs[e]
		if !ok {
			return spec.Spec{}, fmt.Errorf("--vpc-cidr: missing CIDR for env %q (use --vpc-cidr %s=<cidr>)", e, e)
		}
		nat := defaultNATMode(e)
		if raw, ok := natModes[e]; ok {
			nat = spec.NATMode(raw)
		}
		envs = append(envs, spec.Environment{
			Name:    e,
			Account: accountForEnv[e],
			Networking: spec.NetworkingSpec{
				VPCCIDR: cidr,
				NAT:     nat,
			},
		})
	}

	return spec.Spec{Region: opts.region, Accounts: accounts, Environments: envs}, nil
}

// buildAccounts derives the account topology and each environment's account
// assignment from the CLI flags, per the AccountsSpec contract: single mode
// is one account shared by every environment, multi mode is one workload
// account per environment (named after the environment) plus one
// management account.
func buildAccounts(opts initOptions) (spec.AccountsSpec, map[string]string, error) {
	accountForEnv := make(map[string]string, len(opts.envs))

	if opts.singleAccount {
		if opts.managementAccountSet {
			return spec.AccountsSpec{}, nil, errors.New("--management-account is only valid with --multi-account")
		}
		for _, e := range opts.envs {
			accountForEnv[e] = opts.accountName
		}
		return spec.AccountsSpec{
			Mode:     spec.AccountModeSingle,
			Accounts: []spec.Account{{Name: opts.accountName}},
		}, accountForEnv, nil
	}

	if opts.accountNameSet {
		return spec.AccountsSpec{}, nil, errors.New("--account-name is only valid with --single-account")
	}
	if opts.managementAccount == "" {
		return spec.AccountsSpec{}, nil, errors.New("--management-account is required with --multi-account")
	}

	accts := []spec.Account{{Name: opts.managementAccount, IsManagement: true}}
	for _, e := range opts.envs {
		accts = append(accts, spec.Account{Name: e})
		accountForEnv[e] = e
	}
	return spec.AccountsSpec{Mode: spec.AccountModeMulti, Accounts: accts}, accountForEnv, nil
}

// defaultNATMode applies the policy documented on spec.NATMode: single NAT
// for every environment except "prod", which gets one per AZ. --nat
// overrides this per environment.
func defaultNATMode(env string) spec.NATMode {
	if env == "prod" {
		return spec.NATModePerAZ
	}
	return spec.NATModeSingle
}

// parsePerEnvFlag parses repeated "env=value" flag values into a map,
// rejecting malformed pairs, values for environments that were never
// passed via --env, and duplicate entries for the same environment.
func parsePerEnvFlag(flagName string, pairs []string, envs map[string]struct{}) (map[string]string, error) {
	result := make(map[string]string, len(pairs))
	for _, p := range pairs {
		k, v, ok := strings.Cut(p, "=")
		if !ok || k == "" || v == "" {
			return nil, fmt.Errorf("--%s: %q must be in the form env=value", flagName, p)
		}
		if _, known := envs[k]; !known {
			return nil, fmt.Errorf("--%s: env %q was not passed via --env", flagName, k)
		}
		if _, dup := result[k]; dup {
			return nil, fmt.Errorf("--%s: env %q specified more than once", flagName, k)
		}
		result[k] = v
	}
	return result, nil
}
