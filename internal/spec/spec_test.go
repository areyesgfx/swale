package spec_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/areyesgfx/swale/internal/spec"
)

// validMulti returns a valid multi-account spec used as the baseline that
// the validation table tests mutate into invalid variants.
func validMulti() spec.Spec {
	return spec.Spec{
		Accounts: spec.AccountsSpec{
			Mode: spec.AccountModeMulti,
			Accounts: []spec.Account{
				{Name: "mgmt", IsManagement: true},
				{Name: "workloads"},
			},
		},
		Environments: []spec.Environment{
			{Name: "dev", Account: "workloads", Networking: spec.NetworkingSpec{VPCCIDR: "10.0.0.0/16", NAT: spec.NATModeSingle}},
			{Name: "prod", Account: "workloads", Networking: spec.NetworkingSpec{VPCCIDR: "10.1.0.0/16", NAT: spec.NATModePerAZ}},
		},
	}
}

func validSingle() spec.Spec {
	return spec.Spec{
		Accounts: spec.AccountsSpec{
			Mode:     spec.AccountModeSingle,
			Accounts: []spec.Account{{Name: "main"}},
		},
		Environments: []spec.Environment{
			{Name: "dev", Account: "main", Networking: spec.NetworkingSpec{VPCCIDR: "10.0.0.0/16", NAT: spec.NATModeSingle}},
		},
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	cases := map[string]spec.Spec{
		"multi":  validMulti(),
		"single": validSingle(),
	}
	for name, want := range cases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "swale.yaml")
			if err := spec.Save(want, path); err != nil {
				t.Fatalf("Save: %v", err)
			}
			got, err := spec.Load(path)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("round-trip mismatch:\n got: %#v\nwant: %#v", got, want)
			}
		})
	}
}

func TestValidateInvalid(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*spec.Spec)
		wantSub string
	}{
		{"no environments", func(s *spec.Spec) { s.Environments = nil }, "at least one environment"},
		{"empty env name", func(s *spec.Spec) { s.Environments[0].Name = "" }, "environments[0].name"},
		{"duplicate env name", func(s *spec.Spec) { s.Environments[1].Name = "dev" }, "duplicate environment name"},
		{"unknown account ref", func(s *spec.Spec) { s.Environments[0].Account = "nope" }, "does not match any account"},
		{"empty account ref", func(s *spec.Spec) { s.Environments[0].Account = "" }, "account: must not be empty"},
		{"bad mode", func(s *spec.Spec) { s.Accounts.Mode = "sometimes" }, "not a valid account mode"},
		{"empty account name", func(s *spec.Spec) { s.Accounts.Accounts[1].Name = "" }, "accounts.accounts[1].name"},
		{"duplicate account name", func(s *spec.Spec) { s.Accounts.Accounts[1].Name = "mgmt" }, "duplicate account name"},
		{"multi without mgmt", func(s *spec.Spec) { s.Accounts.Accounts[0].IsManagement = false }, "exactly one management account"},
		{"single with two accounts", func(s *spec.Spec) {
			*s = validSingle()
			s.Accounts.Accounts = append(s.Accounts.Accounts, spec.Account{Name: "extra"})
		}, "requires exactly one account"},
		{"single with mgmt", func(s *spec.Spec) {
			*s = validSingle()
			s.Accounts.Accounts[0].IsManagement = true
		}, "must not declare a management account"},
		{"empty cidr", func(s *spec.Spec) { s.Environments[0].Networking.VPCCIDR = "" }, "vpc_cidr: must not be empty"},
		{"malformed cidr", func(s *spec.Spec) { s.Environments[0].Networking.VPCCIDR = "10.0.0.0/33" }, "not valid CIDR notation"},
		{"cidr host bits set", func(s *spec.Spec) { s.Environments[0].Networking.VPCCIDR = "10.0.0.5/16" }, "host bits set"},
		{"ipv6 cidr", func(s *spec.Spec) { s.Environments[0].Networking.VPCCIDR = "2001:db8::/32" }, "must be an IPv4 CIDR"},
		{"bad nat mode", func(s *spec.Spec) { s.Environments[0].Networking.NAT = "double" }, "not a valid NAT mode"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := validMulti()
			tt.mutate(&s)
			err := s.Validate()
			if err == nil {
				t.Fatalf("Validate: want error containing %q, got nil", tt.wantSub)
			}
			if !strings.Contains(err.Error(), tt.wantSub) {
				t.Errorf("Validate error = %q, want it to contain %q", err.Error(), tt.wantSub)
			}
		})
	}
}

func TestValidateReportsAllProblems(t *testing.T) {
	s := validMulti()
	s.Accounts.Mode = "bogus"
	s.Environments[0].Networking.NAT = "bogus"
	err := s.Validate()
	if err == nil {
		t.Fatal("Validate: want error, got nil")
	}
	for _, want := range []string{"not a valid account mode", "not a valid NAT mode"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Validate error = %q, want it to contain %q", err.Error(), want)
		}
	}
}

func TestSaveRefusesInvalidSpec(t *testing.T) {
	s := validMulti()
	s.Environments[0].Networking.VPCCIDR = "not-a-cidr"
	path := filepath.Join(t.TempDir(), "swale.yaml")
	if err := spec.Save(s, path); err == nil {
		t.Fatal("Save: want error for invalid spec, got nil")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("Save wrote a file for an invalid spec (stat err = %v)", err)
	}
}

func TestLoadRejectsUnknownField(t *testing.T) {
	path := filepath.Join(t.TempDir(), "swale.yaml")
	body := "accounts:\n  mode: single\n  accounts:\n    - name: main\ntypo_field: oops\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, err := spec.Load(path)
	if err == nil {
		t.Fatal("Load: want error for unknown field, got nil")
	}
	if !strings.Contains(err.Error(), "typo_field") {
		t.Errorf("Load error = %q, want it to mention the unknown field", err.Error())
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := spec.Load(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err == nil {
		t.Fatal("Load: want error for missing file, got nil")
	}
}
