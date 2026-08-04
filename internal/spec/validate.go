package spec

import (
	"errors"
	"fmt"
	"net"
)

// Validate reports whether a Spec is internally consistent and safe to
// generate from. It collects every problem it finds and returns them
// joined together (errors.Join), so callers see all of them at once rather
// than fixing one, re-running, and hitting the next.
func (s Spec) Validate() error {
	var errs []error

	errs = append(errs, s.Accounts.validate()...)

	if len(s.Environments) == 0 {
		errs = append(errs, errors.New("environments: at least one environment is required"))
	}

	// Environments reference accounts by name, so index the declared names.
	accountNames := make(map[string]struct{}, len(s.Accounts.Accounts))
	for _, a := range s.Accounts.Accounts {
		accountNames[a.Name] = struct{}{}
	}

	seen := make(map[string]struct{}, len(s.Environments))
	for i, env := range s.Environments {
		where := fmt.Sprintf("environments[%d]", i)

		if env.Name == "" {
			errs = append(errs, fmt.Errorf("%s.name: must not be empty", where))
		} else {
			if _, dup := seen[env.Name]; dup {
				errs = append(errs, fmt.Errorf("%s.name: duplicate environment name %q", where, env.Name))
			}
			seen[env.Name] = struct{}{}
		}

		if env.Account == "" {
			errs = append(errs, fmt.Errorf("%s.account: must not be empty", where))
		} else if _, ok := accountNames[env.Account]; !ok {
			errs = append(errs, fmt.Errorf("%s.account: %q does not match any account in accounts.accounts", where, env.Account))
		}

		errs = append(errs, env.Networking.validate(where)...)
	}

	return errors.Join(errs...)
}

func (a AccountsSpec) validate() []error {
	var errs []error

	switch a.Mode {
	case AccountModeSingle, AccountModeMulti:
	case "":
		errs = append(errs, errors.New(`accounts.mode: must not be empty (want "single" or "multi")`))
	default:
		errs = append(errs, fmt.Errorf(`accounts.mode: %q is not a valid account mode (want "single" or "multi")`, a.Mode))
	}

	if len(a.Accounts) == 0 {
		errs = append(errs, errors.New("accounts.accounts: at least one account is required"))
	}

	seen := make(map[string]struct{}, len(a.Accounts))
	management := 0
	for i, acct := range a.Accounts {
		if acct.Name == "" {
			errs = append(errs, fmt.Errorf("accounts.accounts[%d].name: must not be empty", i))
		} else {
			if _, dup := seen[acct.Name]; dup {
				errs = append(errs, fmt.Errorf("accounts.accounts[%d].name: duplicate account name %q", i, acct.Name))
			}
			seen[acct.Name] = struct{}{}
		}
		if acct.IsManagement {
			management++
		}
	}

	// Topology rules follow the AccountsSpec contract: single-account mode is
	// one workload account with no separate management account; multi-account
	// mode is exactly one management account plus at least one workload account.
	switch a.Mode {
	case AccountModeSingle:
		if len(a.Accounts) != 1 {
			errs = append(errs, fmt.Errorf("accounts.accounts: single-account mode requires exactly one account, got %d", len(a.Accounts)))
		}
		if management != 0 {
			errs = append(errs, errors.New("accounts.accounts: single-account mode must not declare a management account"))
		}
	case AccountModeMulti:
		if management != 1 {
			errs = append(errs, fmt.Errorf("accounts.accounts: multi-account mode requires exactly one management account, got %d", management))
		}
		if len(a.Accounts) < 2 {
			errs = append(errs, fmt.Errorf("accounts.accounts: multi-account mode requires a management account plus at least one workload account, got %d", len(a.Accounts)))
		}
	}

	return errs
}

func (n NetworkingSpec) validate(where string) []error {
	var errs []error

	if n.VPCCIDR == "" {
		errs = append(errs, fmt.Errorf("%s.networking.vpc_cidr: must not be empty", where))
	} else {
		ip, ipNet, err := net.ParseCIDR(n.VPCCIDR)
		switch {
		case err != nil:
			errs = append(errs, fmt.Errorf("%s.networking.vpc_cidr: %q is not valid CIDR notation: %w", where, n.VPCCIDR, err))
		case ipNet.IP.To4() == nil:
			errs = append(errs, fmt.Errorf("%s.networking.vpc_cidr: %q must be an IPv4 CIDR", where, n.VPCCIDR))
		case !ip.Equal(ipNet.IP):
			errs = append(errs, fmt.Errorf("%s.networking.vpc_cidr: %q has host bits set; use the network address %s", where, n.VPCCIDR, ipNet.String()))
		}
	}

	switch n.NAT {
	case NATModeSingle, NATModePerAZ:
	case "":
		errs = append(errs, fmt.Errorf(`%s.networking.nat: must not be empty (want "single" or "per_az")`, where))
	default:
		errs = append(errs, fmt.Errorf(`%s.networking.nat: %q is not a valid NAT mode (want "single" or "per_az")`, where, n.NAT))
	}

	return errs
}
