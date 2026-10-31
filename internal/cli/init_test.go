package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"

	"github.com/areyesgfx/swale/internal/spec"
)

// run executes cmd with args and returns what it printed to stdout.
func run(t *testing.T, cmd *cobra.Command, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

// runInit runs "swale init" with args in a fresh project directory, which
// it returns along with the command's output.
func runInit(t *testing.T, args ...string) (dir, out string, err error) {
	t.Helper()
	dir = t.TempDir()
	out, err = run(t, newInitCmd(), append(args, dir)...)
	return dir, out, err
}

var validInitFlags = []string{
	"--region", "us-east-1",
	"--env", "dev",
	"--env", "prod",
	"--single-account",
	"--vpc-cidr", "dev=10.0.0.0/16",
	"--vpc-cidr", "prod=10.1.0.0/16",
}

func TestInitFlagsSingleAccount(t *testing.T) {
	dir, out, err := runInit(t, validInitFlags...)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	got, err := spec.Load(filepath.Join(dir, spec.FileName))
	if err != nil {
		t.Fatalf("spec.Load: %v", err)
	}
	want := spec.Spec{
		Region: "us-east-1",
		Accounts: spec.AccountsSpec{
			Mode:     spec.AccountModeSingle,
			Accounts: []spec.Account{{Name: "main"}},
		},
		Environments: []spec.Environment{
			{Name: "dev", Account: "main", Networking: spec.NetworkingSpec{VPCCIDR: "10.0.0.0/16", NAT: spec.NATModeSingle}},
			{Name: "prod", Account: "main", Networking: spec.NetworkingSpec{VPCCIDR: "10.1.0.0/16", NAT: spec.NATModePerAZ}},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("loaded spec = %#v, want %#v", got, want)
	}

	if _, err := os.Stat(filepath.Join(dir, "live/main/dev/terragrunt.stack.hcl")); err != nil {
		t.Errorf("init did not generate the project: %v", err)
	}
	for _, want := range []string{"created  root.hcl", "terragrunt stack run apply --backend-bootstrap"} {
		if !strings.Contains(out, want) {
			t.Errorf("init output missing %q:\n%s", want, out)
		}
	}
}

func TestInitRejectsMultiAccountBeforeWriting(t *testing.T) {
	dir, _, err := runInit(t,
		"--region", "us-east-1",
		"--env", "dev",
		"--multi-account",
		"--management-account", "mgmt",
		"--vpc-cidr", "dev=10.0.0.0/16",
	)
	if err == nil || !strings.Contains(err.Error(), "single-account") {
		t.Fatalf("Execute error = %v, want one mentioning single-account", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("init left %d entries in the project directory, want none", len(entries))
	}
}

func TestInitRefusesExistingSpec(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, spec.FileName)
	if err := os.WriteFile(specPath, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := run(t, newInitCmd(), append(validInitFlags, dir)...)
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("Execute error = %v, want one saying the spec already exists", err)
	}
	if got, _ := os.ReadFile(specPath); string(got) != "mine" {
		t.Errorf("existing spec was overwritten with %q", got)
	}
}

func TestInitFlagsMultiAccountAnswers(t *testing.T) {
	f := initFlags{
		region:            "us-east-1",
		envs:              []string{"dev", "prod"},
		multiAccount:      true,
		managementAccount: "mgmt",
		vpcCIDRs:          []string{"dev=10.0.0.0/16", "prod=10.1.0.0/16"},
		natModes:          []string{"dev=per_az"},
	}
	a, err := f.answers()
	if err != nil {
		t.Fatalf("answers: %v", err)
	}
	got := a.toSpec()
	want := spec.Spec{
		Region: "us-east-1",
		Accounts: spec.AccountsSpec{
			Mode: spec.AccountModeMulti,
			Accounts: []spec.Account{
				{Name: "mgmt", IsManagement: true},
				{Name: "dev"},
				{Name: "prod"},
			},
		},
		Environments: []spec.Environment{
			{Name: "dev", Account: "dev", Networking: spec.NetworkingSpec{VPCCIDR: "10.0.0.0/16", NAT: spec.NATModePerAZ}},
			{Name: "prod", Account: "prod", Networking: spec.NetworkingSpec{VPCCIDR: "10.1.0.0/16", NAT: spec.NATModePerAZ}},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("loaded spec = %#v, want %#v", got, want)
	}
}

func TestInitFlagErrors(t *testing.T) {
	base := []string{"--region", "us-east-1", "--env", "dev", "--vpc-cidr", "dev=10.0.0.0/16"}
	tests := []struct {
		name    string
		args    []string
		wantSub string
	}{
		{"no account mode", base, "one of --single-account or --multi-account"},
		{"both account modes", append(base, "--single-account", "--multi-account"), "none of the others can be"},
		{"management account with single", append(base, "--single-account", "--management-account", "mgmt"), "none of the others can be"},
		{"account name with multi", append(base, "--multi-account", "--account-name", "x"), "none of the others can be"},
		{"multi without management account", append(base, "--multi-account"), "--management-account is required with --multi-account"},
		{"missing region", []string{"--env", "dev", "--single-account", "--vpc-cidr", "dev=10.0.0.0/16"}, "region: must not be empty"},
		{"missing cidr", []string{"--region", "us-east-1", "--env", "dev", "--single-account"}, "vpc_cidr: must not be empty"},
		{"invalid cidr", []string{"--region", "us-east-1", "--env", "dev", "--single-account", "--vpc-cidr", "dev=nope"}, "not valid CIDR notation"},
		{"duplicate env", append(base, "--env", "dev", "--single-account"), "duplicate environment name"},
		{"malformed pair", append(base, "--single-account", "--nat", "dev"), "must be in the form env=value"},
		{"pair for unknown env", append(base, "--single-account", "--vpc-cidr", "prod=10.1.0.0/16"), `env "prod" was not passed via --env`},
		{"duplicate pair", append(base, "--single-account", "--vpc-cidr", "dev=10.1.0.0/16"), "specified more than once"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, _, err := runInit(t, tt.args...)
			if err == nil {
				t.Fatalf("Execute: want error containing %q, got nil", tt.wantSub)
			}
			if !strings.Contains(err.Error(), tt.wantSub) {
				t.Errorf("Execute error = %q, want it to contain %q", err.Error(), tt.wantSub)
			}
			if _, err := os.Stat(filepath.Join(dir, spec.FileName)); err == nil {
				t.Error("a spec was written despite the error")
			}
		})
	}
}

// TestWizardForms drives the real huh forms through their accessible
// (line-based) mode, so the test needs no terminal. Each huh field reads
// stdin through its own buffered scanner, so input goes through
// OneByteReader to stop one field from swallowing the next field's line.
func TestWizardForms(t *testing.T) {
	var (
		a       initAnswers
		envList string
	)
	accounts := "us-west-2\n" + // region
		"2\n" + // account layout: multi
		"mgmt\n" + // management account
		"dev, staging\n" // environments
	err := accountsForm(&a, &envList).
		WithAccessible(true).
		WithInput(iotest.OneByteReader(strings.NewReader(accounts))).
		WithOutput(io.Discard).
		Run()
	if err != nil {
		t.Fatalf("accounts form: %v", err)
	}

	a.envs = envsFromList(envList)
	// Accessible mode doesn't offer pre-filled values as defaults (a blank
	// line is an empty answer and fails validation), so type each value.
	networking := "10.0.0.0/16\n" + // dev CIDR
		"1\n" + // dev NAT: single
		"10.20.0.0/16\n" + // staging CIDR
		"2\n" // staging NAT: per_az
	err = networkingForm(a.envs).
		WithAccessible(true).
		WithInput(iotest.OneByteReader(strings.NewReader(networking))).
		WithOutput(io.Discard).
		Run()
	if err != nil {
		t.Fatalf("networking form: %v", err)
	}

	want := spec.Spec{
		Region: "us-west-2",
		Accounts: spec.AccountsSpec{
			Mode: spec.AccountModeMulti,
			Accounts: []spec.Account{
				{Name: "mgmt", IsManagement: true},
				{Name: "dev"},
				{Name: "staging"},
			},
		},
		Environments: []spec.Environment{
			{Name: "dev", Account: "dev", Networking: spec.NetworkingSpec{VPCCIDR: "10.0.0.0/16", NAT: spec.NATModeSingle}},
			{Name: "staging", Account: "staging", Networking: spec.NetworkingSpec{VPCCIDR: "10.20.0.0/16", NAT: spec.NATModePerAZ}},
		},
	}
	if got := a.toSpec(); !reflect.DeepEqual(got, want) {
		t.Errorf("wizard spec = %#v, want %#v", got, want)
	}
}

func TestCheckEnvList(t *testing.T) {
	for list, wantErr := range map[string]bool{
		"dev, prod":     false,
		" dev ,,prod, ": false,
		"":              true,
		" , ":           true,
		"dev, dev":      true,
	} {
		if err := checkEnvList(list); (err != nil) != wantErr {
			t.Errorf("checkEnvList(%q) error = %v, want error: %v", list, err, wantErr)
		}
	}
}

// TestBackableInput pins the huh v2.0.3 workaround: an invalid input must
// not block leaving the screen, but must still report its error once it
// has focus again.
func TestBackableInput(t *testing.T) {
	in := &backableInput{Input: huh.NewInput().Validate(notEmpty)}
	in.Focus()
	in.Blur()
	if err := in.Error(); err != nil {
		t.Errorf("Error() while leaving = %v, want nil", err)
	}
	in.Focus()
	if err := in.Error(); err == nil {
		t.Error("Error() after refocus = nil, want the validation error")
	}
}
