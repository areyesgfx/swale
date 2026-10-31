// Package spec defines Spec, the intermediate representation both of
// swale's front doors (wizard and CLI flags) produce, and that
// internal/generate consumes to emit Terraform/Terragrunt (see ADR-0001).
// Compute, storage, and secrets sections are added by later phases once
// their shape is decided, rather than stubbed out here ahead of time.
package spec

// Spec is the root of a swale.yaml file. v1 supports exactly one region
// per project.
type Spec struct {
	Region       string        `yaml:"region"`
	Accounts     AccountsSpec  `yaml:"accounts"`
	Environments []Environment `yaml:"environments"`
}

// AccountsSpec describes the AWS account topology. In AccountModeSingle,
// Accounts holds exactly one entry. In AccountModeMulti, it holds one
// workload account per environment plus one management account.
type AccountsSpec struct {
	Mode     AccountMode `yaml:"mode"`
	Accounts []Account   `yaml:"accounts"`
}

type AccountMode string

const (
	AccountModeSingle AccountMode = "single"
	AccountModeMulti  AccountMode = "multi"
)

// Account is an AWS account folder under live/<account>/.
type Account struct {
	Name         string `yaml:"name"`
	IsManagement bool   `yaml:"is_management"`
}

// Environment is a single deployment environment, e.g. "dev" or "prod".
// Account must match the Name of an entry in AccountsSpec.Accounts.
type Environment struct {
	Name       string         `yaml:"name"`
	Account    string         `yaml:"account"`
	Networking NetworkingSpec `yaml:"networking"`
}

type NetworkingSpec struct {
	VPCCIDR string  `yaml:"vpc_cidr"`
	NAT     NATMode `yaml:"nat"`
}

type NATMode string

const (
	NATModeSingle NATMode = "single"
	NATModePerAZ  NATMode = "per_az"
)

// DefaultNAT is the NAT policy both front doors apply when the user
// doesn't choose: one NAT gateway per AZ for "prod", a single shared one
// everywhere else.
func DefaultNAT(env string) NATMode {
	if env == "prod" {
		return NATModePerAZ
	}
	return NATModeSingle
}
