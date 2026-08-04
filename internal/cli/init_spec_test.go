package cli

import (
	"reflect"
	"strings"
	"testing"

	"github.com/areyesgfx/swale/internal/spec"
)

func TestBuildSpecSingleAccount(t *testing.T) {
	opts := initOptions{
		region:        "us-east-1",
		envs:          []string{"dev", "prod"},
		singleAccount: true,
		accountName:   "main",
		vpcCIDRs:      []string{"dev=10.0.0.0/16", "prod=10.1.0.0/16"},
	}
	got, err := buildSpec(opts)
	if err != nil {
		t.Fatalf("buildSpec: %v", err)
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
		t.Errorf("buildSpec = %#v, want %#v", got, want)
	}
}

func TestBuildSpecMultiAccount(t *testing.T) {
	opts := initOptions{
		region:            "us-east-1",
		envs:              []string{"dev", "prod"},
		multiAccount:      true,
		managementAccount: "mgmt",
		vpcCIDRs:          []string{"dev=10.0.0.0/16", "prod=10.1.0.0/16"},
	}
	got, err := buildSpec(opts)
	if err != nil {
		t.Fatalf("buildSpec: %v", err)
	}
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
			{Name: "dev", Account: "dev", Networking: spec.NetworkingSpec{VPCCIDR: "10.0.0.0/16", NAT: spec.NATModeSingle}},
			{Name: "prod", Account: "prod", Networking: spec.NetworkingSpec{VPCCIDR: "10.1.0.0/16", NAT: spec.NATModePerAZ}},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildSpec = %#v, want %#v", got, want)
	}
}

func TestBuildSpecNATOverride(t *testing.T) {
	opts := initOptions{
		region:        "us-east-1",
		envs:          []string{"dev"},
		singleAccount: true,
		accountName:   "main",
		vpcCIDRs:      []string{"dev=10.0.0.0/16"},
		natModes:      []string{"dev=per_az"},
	}
	got, err := buildSpec(opts)
	if err != nil {
		t.Fatalf("buildSpec: %v", err)
	}
	if got.Environments[0].Networking.NAT != spec.NATModePerAZ {
		t.Errorf("NAT = %q, want %q", got.Environments[0].Networking.NAT, spec.NATModePerAZ)
	}
}

func TestBuildSpecErrors(t *testing.T) {
	base := func() initOptions {
		return initOptions{
			region:        "us-east-1",
			envs:          []string{"dev"},
			singleAccount: true,
			accountName:   "main",
			vpcCIDRs:      []string{"dev=10.0.0.0/16"},
		}
	}

	tests := []struct {
		name    string
		mutate  func(*initOptions)
		wantSub string
	}{
		{"missing region", func(o *initOptions) { o.region = "" }, "--region is required"},
		{"neither mode", func(o *initOptions) { o.singleAccount = false }, "exactly one of --single-account or --multi-account"},
		{"both modes", func(o *initOptions) { o.multiAccount = true }, "exactly one of --single-account or --multi-account"},
		{"no envs", func(o *initOptions) { o.envs = nil }, "--env is required"},
		{"empty env name", func(o *initOptions) { o.envs = []string{""} }, "must not be empty"},
		{"duplicate env", func(o *initOptions) { o.envs = []string{"dev", "dev"}; o.vpcCIDRs = []string{"dev=10.0.0.0/16"} }, "specified more than once"},
		{"missing cidr", func(o *initOptions) { o.vpcCIDRs = nil }, "missing CIDR for env"},
		{"malformed cidr pair", func(o *initOptions) { o.vpcCIDRs = []string{"dev"} }, "must be in the form env=value"},
		{"cidr for unknown env", func(o *initOptions) { o.vpcCIDRs = []string{"dev=10.0.0.0/16", "prod=10.1.0.0/16"} }, `env "prod" was not passed via --env`},
		{"duplicate cidr for env", func(o *initOptions) {
			o.envs = []string{"dev"}
			o.vpcCIDRs = []string{"dev=10.0.0.0/16", "dev=10.1.0.0/16"}
		}, "specified more than once"},
		{"management-account with single", func(o *initOptions) { o.managementAccount = "mgmt"; o.managementAccountSet = true }, "--management-account is only valid with --multi-account"},
		{"account-name with multi", func(o *initOptions) {
			o.singleAccount = false
			o.multiAccount = true
			o.managementAccount = "mgmt"
			o.accountNameSet = true
		}, "--account-name is only valid with --single-account"},
		{"multi without management account", func(o *initOptions) {
			o.singleAccount = false
			o.multiAccount = true
		}, "--management-account is required with --multi-account"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := base()
			tt.mutate(&opts)
			_, err := buildSpec(opts)
			if err == nil {
				t.Fatalf("buildSpec: want error containing %q, got nil", tt.wantSub)
			}
			if !strings.Contains(err.Error(), tt.wantSub) {
				t.Errorf("buildSpec error = %q, want it to contain %q", err.Error(), tt.wantSub)
			}
		})
	}
}
