// Package spec defines Spec, the single intermediate representation that
// both of swale's front doors (the interactive wizard and direct CLI flags)
// produce, and that internal/generate consumes to emit Terraform/Terragrunt.
// See docs/adr/0001-generate-with-text-template.md for why generation is a
// pure function from a Spec to files, rather than each front door emitting
// HCL directly.
//
// This is a Phase 1 struct: it covers environments, account topology, and
// networking only. Compute, storage, and secrets sections are added by
// later phases once their shape is actually decided (see the project build
// plan) rather than stubbed out here ahead of time.
package spec

// Spec is the root of a swale.yaml file.
type Spec struct {
	Accounts     AccountsSpec
	Environments []Environment
}
