package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/charmbracelet/x/term"

	"github.com/areyesgfx/swale/internal/spec"
)

// regions is the wizard's shortlist, so nobody has to know region codes.
// It's a convenience, not a limit: --region accepts any region.
var regions = []struct{ place, code string }{
	{"US East (N. Virginia)", "us-east-1"},
	{"US East (Ohio)", "us-east-2"},
	{"US West (Oregon)", "us-west-2"},
	{"Canada (Central)", "ca-central-1"},
	{"South America (São Paulo)", "sa-east-1"},
	{"Europe (Ireland)", "eu-west-1"},
	{"Europe (London)", "eu-west-2"},
	{"Europe (Frankfurt)", "eu-central-1"},
	{"Asia Pacific (Mumbai)", "ap-south-1"},
	{"Asia Pacific (Singapore)", "ap-southeast-1"},
	{"Asia Pacific (Tokyo)", "ap-northeast-1"},
	{"Asia Pacific (Sydney)", "ap-southeast-2"},
}

// runWizard is the interactive "swale init" front door. defaults holds the
// pre-filled answers, so pressing enter on every question is a sane setup.
func runWizard(defaults initAnswers) (initAnswers, error) {
	if !term.IsTerminal(os.Stdin.Fd()) {
		return initAnswers{}, errors.New("stdin is not a terminal, so the interactive questions can't run; pass flags instead (see swale init --help)")
	}
	a := defaults
	if err := initForm(&a).Run(); err != nil {
		return initAnswers{}, err
	}
	return a, nil
}

func initForm(a *initAnswers) *huh.Form {
	regionOptions := make([]huh.Option[string], len(regions))
	for i, r := range regions {
		regionOptions[i] = huh.NewOption(fmt.Sprintf("%-26s %s", r.place, r.code), r.code)
	}

	var envOptions []huh.Option[int]
	for n := 1; n <= len(spec.KnownEnvironments); n++ {
		envs, _ := spec.EnvironmentsFor(n)
		envOptions = append(envOptions, huh.NewOption(fmt.Sprintf("%d: %s", n, joinAnd(envs)), n))
	}

	return oneQuestionPerScreen(
		huh.NewInput().
			Title("What's your app called?").
			Description("Lowercase letters, numbers and dashes, like acme-api. swale names your AWS resources after it.").
			Value(&a.name).
			Validate(spec.CheckName),
		huh.NewSelect[string]().
			Title("Where should it run?").
			Description("Pick the AWS region closest to most of your users.").
			Options(regionOptions...).
			Value(&a.region),
		huh.NewSelect[int]().
			Title("How many environments do you need?").
			Description("Each one is a separate copy of your infrastructure. prod keeps running if a data center fails; the others use a cheaper setup.").
			Options(envOptions...).
			Value(&a.environments),
	)
}

// joinAnd lists items the way a sentence would: "a", "a and b", "a, b and c".
func joinAnd(items []string) string {
	if len(items) <= 1 {
		return strings.Join(items, "")
	}
	return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
}

// oneQuestionPerScreen puts each field in its own huh group, since a
// group is what huh renders as one screen.
func oneQuestionPerScreen(fields ...huh.Field) *huh.Form {
	groups := make([]*huh.Group, len(fields))
	for i, f := range fields {
		if in, ok := f.(*huh.Input); ok {
			f = &backableInput{Input: in}
		}
		groups[i] = huh.NewGroup(f)
	}
	return huh.NewForm(groups...)
}

// backableInput works around a huh v2.0.3 bug: shift+tab on an invalid
// input blurs it (which validates), then the form refuses to leave the
// screen because of the error but never refocuses the field, so the form
// is stuck until ctrl+c. Hiding the error only while the field is being
// left (Blur until the next Focus) lets the user go back. Enter still
// validates and shows the error before moving forward, since huh checks
// that inside Input.Update before blurring, and spec.Validate checks
// everything at the end.
type backableInput struct {
	*huh.Input
	leaving bool
}

func (b *backableInput) Blur() tea.Cmd {
	b.leaving = true
	return b.Input.Blur()
}

func (b *backableInput) Focus() tea.Cmd {
	b.leaving = false
	return b.Input.Focus()
}

func (b *backableInput) Error() error {
	if b.leaving {
		return nil
	}
	return b.Input.Error()
}

func (b *backableInput) Update(msg tea.Msg) (huh.Model, tea.Cmd) {
	_, cmd := b.Input.Update(msg)
	return b, cmd
}
