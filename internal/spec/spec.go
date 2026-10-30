// Package spec defines Spec, the intermediate representation both of
// swale's front doors (wizard and CLI flags) produce, and that
// internal/generate consumes to emit Terraform/Terragrunt (see ADR-0001).
// Compute, storage, and secrets sections are added by later phases once
// their shape is decided, rather than stubbed out here ahead of time.
package spec

// Spec is the root of a swale.yaml file.
type Spec struct {
	// Region is the AWS region generated infrastructure is deployed into.
	// v1 supports exactly one region per project.
	Region       string        `yaml:"region"`
	Accounts     AccountsSpec  `yaml:"accounts"`
	Environments []Environment `yaml:"environments"`
}
