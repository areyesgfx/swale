package spec

import (
	"errors"
	"fmt"
	"net"
	"regexp"
)

// regionPattern is a format check (e.g. "us-east-1", "us-gov-west-1"), not
// an exhaustive allow-list - AWS adds regions over time, so validating
// against a hardcoded list would rot.
var regionPattern = regexp.MustCompile(`^[a-z]{2}(-[a-z]+)+-\d+$`)

// Validate reports whether a Spec is internally consistent and safe to
// generate from. It collects every problem it finds and returns them
// joined together (errors.Join), so callers see all of them at once rather
// than fixing one, re-running, and hitting the next.
func (s Spec) Validate() error {
	var errs []error

	if err := CheckRegion(s.Region); err != nil {
		errs = append(errs, fmt.Errorf("region: %w", err))
	}

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

	if err := CheckCIDR(n.VPCCIDR); err != nil {
		errs = append(errs, fmt.Errorf("%s.networking.vpc_cidr: %w", where, err))
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

// CheckRegion is the per-field region rule Validate applies, exported so
// the wizard can reject a bad value as it's typed instead of after the
// whole form.
func CheckRegion(region string) error {
	if region == "" {
		return errors.New("must not be empty")
	}
	if !regionPattern.MatchString(region) {
		return fmt.Errorf("%q does not look like an AWS region (e.g. \"us-east-1\")", region)
	}
	return nil
}

// CheckCIDR is the per-field VPC CIDR rule Validate applies, exported for
// the same reason as CheckRegion.
func CheckCIDR(cidr string) error {
	if cidr == "" {
		return errors.New("must not be empty")
	}
	ip, ipNet, err := net.ParseCIDR(cidr)
	switch {
	case err != nil:
		return fmt.Errorf("%q is not valid CIDR notation: %w", cidr, err)
	case ipNet.IP.To4() == nil:
		return fmt.Errorf("%q must be an IPv4 CIDR", cidr)
	case !ip.Equal(ipNet.IP):
		return fmt.Errorf("%q has host bits set; use the network address %s", cidr, ipNet)
	}
	return nil
}
