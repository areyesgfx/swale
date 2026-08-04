package cli

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/areyesgfx/swale/internal/spec"
)

func TestInitCommandSingleAccount(t *testing.T) {
	out := filepath.Join(t.TempDir(), "swale.yaml")

	cmd := newInitCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"--env", "dev",
		"--single-account",
		"--vpc-cidr", "dev=10.0.0.0/16",
		"--output", out,
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	got, err := spec.Load(out)
	if err != nil {
		t.Fatalf("spec.Load: %v", err)
	}
	want := spec.Spec{
		Accounts: spec.AccountsSpec{
			Mode:     spec.AccountModeSingle,
			Accounts: []spec.Account{{Name: "main"}},
		},
		Environments: []spec.Environment{
			{Name: "dev", Account: "main", Networking: spec.NetworkingSpec{VPCCIDR: "10.0.0.0/16", NAT: spec.NATModeSingle}},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("loaded spec = %#v, want %#v", got, want)
	}
}

func TestInitCommandMultiAccount(t *testing.T) {
	out := filepath.Join(t.TempDir(), "swale.yaml")

	cmd := newInitCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"--env", "dev",
		"--env", "prod",
		"--multi-account",
		"--management-account", "mgmt",
		"--vpc-cidr", "dev=10.0.0.0/16",
		"--vpc-cidr", "prod=10.1.0.0/16",
		"--output", out,
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	got, err := spec.Load(out)
	if err != nil {
		t.Fatalf("spec.Load: %v", err)
	}
	want := spec.Spec{
		Accounts: spec.AccountsSpec{
			Mode: spec.AccountModeMulti,
			Accounts: []spec.Account{
				{Name: "mgmt", IsManagement: true},
				{Name: "dev"},
				{Name: "prod"},
			},
		},
		Environments: []spec.Environment{
			{Name: "dev", Account: "dev", Networking: spec.NetworkingSpec{VPCCIDR: "10.0.0.0/16", NAT: spec.NATModeSingle}},
			{Name: "prod", Account: "prod", Networking: spec.NetworkingSpec{VPCCIDR: "10.1.0.0/16", NAT: spec.NATModePerAZ}},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("loaded spec = %#v, want %#v", got, want)
	}
}

func TestInitCommandInvalidSpecDoesNotWrite(t *testing.T) {
	out := filepath.Join(t.TempDir(), "swale.yaml")

	cmd := newInitCmd()
	cmd.SetOut(&bytes.Buffer{})
	errBuf := &bytes.Buffer{}
	cmd.SetErr(errBuf)
	cmd.SetArgs([]string{
		"--env", "dev",
		"--single-account",
		"--vpc-cidr", "dev=not-a-cidr",
		"--output", out,
	})
	if err := cmd.Execute(); err == nil {
		t.Fatal("Execute: want error for invalid CIDR, got nil")
	}
	if !strings.Contains(errBuf.String(), "vpc_cidr") {
		t.Errorf("stderr = %q, want it to mention vpc_cidr", errBuf.String())
	}
	if _, err := spec.Load(out); err == nil {
		t.Fatal("spec.Load: expected no file to have been written")
	}
}

func TestInitCommandFlagValidationError(t *testing.T) {
	cmd := newInitCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--env", "dev"}) // neither --single-account nor --multi-account
	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute: want error, got nil")
	}
	if !strings.Contains(err.Error(), "--single-account or --multi-account") {
		t.Errorf("Execute error = %q, want it to mention the account mode flags", err.Error())
	}
}
