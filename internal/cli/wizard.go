package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"charm.land/huh/v2"
	"github.com/charmbracelet/x/term"

	"github.com/areyesgfx/swale/internal/spec"
)

// runWizard is the interactive "swale init" front door. It runs two forms
// because the per-environment questions depend on the environment names
// typed into the first one.
func runWizard() (initAnswers, error) {
	if !term.IsTerminal(os.Stdin.Fd()) {
		return initAnswers{}, errors.New("stdin is not a terminal, so the interactive form can't run; pass flags instead (see swale init --help)")
	}

	var (
		a       initAnswers
		envList string
	)
	if err := accountsForm(&a, &envList).Run(); err != nil {
		return initAnswers{}, err
	}
	a.envs = envsFromList(envList)
	if err := networkingForm(a.envs).Run(); err != nil {
		return initAnswers{}, err
	}
	return a, nil
}

// accountsForm asks one account-name question whose title follows the
// layout choice, rather than hiding one of two questions: huh's accessible
// (screen reader) mode ignores hidden groups and would ask both.
func accountsForm(a *initAnswers, envList *string) *huh.Form {
	a.region = "us-east-1"
	*envList = "dev, prod"

	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("AWS region").Value(&a.region).Validate(spec.CheckRegion),
			huh.NewSelect[bool]().Title("Account layout").Options(
				huh.NewOption("Single account shared by every environment", false),
				huh.NewOption("One account per environment, plus a management account", true),
			).Value(&a.multiAccount),
			huh.NewInput().TitleFunc(func() string {
				if a.multiAccount {
					return "Management account name"
				}
				return "Account name"
			}, &a.multiAccount).Placeholder("e.g. main").Value(&a.accountName).Validate(notEmpty),
			huh.NewInput().Title("Environments").Description("Comma-separated, e.g. dev, staging, prod").Value(envList).Validate(checkEnvList),
		),
	)
}

// networkingForm asks for each environment's VPC CIDR and NAT mode,
// writing straight into envs. It pre-fills non-overlapping 10.N.0.0/16
// CIDRs and the default NAT policy so pressing enter is a sane answer.
func networkingForm(envs []envAnswers) *huh.Form {
	groups := make([]*huh.Group, len(envs))
	for i := range envs {
		e := &envs[i]
		e.vpcCIDR = fmt.Sprintf("10.%d.0.0/16", i)
		e.nat = spec.DefaultNAT(e.name)
		groups[i] = huh.NewGroup(
			huh.NewInput().Title(e.name+": VPC CIDR").Value(&e.vpcCIDR).Validate(spec.CheckCIDR),
			huh.NewSelect[spec.NATMode]().Title(e.name+": NAT gateways").Options(
				huh.NewOption("single - one shared NAT gateway, cheaper", spec.NATModeSingle),
				huh.NewOption("per_az - one per availability zone, highly available", spec.NATModePerAZ),
			).Value(&e.nat),
		)
	}
	return huh.NewForm(groups...)
}

func envsFromList(list string) []envAnswers {
	var envs []envAnswers
	for _, name := range strings.Split(list, ",") {
		if name = strings.TrimSpace(name); name != "" {
			envs = append(envs, envAnswers{name: name})
		}
	}
	return envs
}

func checkEnvList(list string) error {
	envs := envsFromList(list)
	if len(envs) == 0 {
		return errors.New("enter at least one environment")
	}
	seen := make(map[string]bool, len(envs))
	for _, e := range envs {
		if seen[e.name] {
			return fmt.Errorf("%q is listed more than once", e.name)
		}
		seen[e.name] = true
	}
	return nil
}

func notEmpty(s string) error {
	if strings.TrimSpace(s) == "" {
		return errors.New("must not be empty")
	}
	return nil
}
