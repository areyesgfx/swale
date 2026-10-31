// Package templates holds the text/template sources swale renders into
// generated Terragrunt/Terraform projects. It uses text/template rather
// than hclwrite so generated files stay plain, readable, and hand-editable;
// the trade-off is that swale can render HCL but never parse it.
package templates

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"
)

//go:embed *.tmpl
var sources embed.FS

var tmpl = template.Must(template.ParseFS(sources, "*.tmpl"))

// NetworkingUnitHCL is units/networking/terragrunt.hcl. Nothing in it
// depends on a Spec (its paths are fixed by the live/<env>/ layout), so
// it's embedded verbatim instead of rendered.
//
//go:embed networking_unit.hcl
var NetworkingUnitHCL string

// GitIgnore is the generated project's .gitignore. The source file has no
// leading dot so git doesn't also apply it to this directory.
//
//go:embed gitignore
var GitIgnore string

// RootData is the input to root.hcl. Terragrunt creates StateBucket when
// run with --backend-bootstrap, so it need not exist yet. Region applies to
// both the state backend and the provider block. Tags are merged into
// default_tags alongside the always-present ManagedBy=swale tag.
type RootData struct {
	StateBucket string
	Region      string
	Tags        map[string]string
}

// EnvironmentData is the infrastructure swale derived for one environment,
// e.g. Name "prod" and ResourceName "acme-api-prod". It's the input to
// live/<env>/terragrunt.stack.hcl, and the README describes each one.
type EnvironmentData struct {
	Name         string
	ResourceName string
	VPCCIDR      string
	NATMode      string
	NATGateways  int
}

// ReadmeData is the input to the generated project's README.md.
// Environments are in the order the user listed them.
type ReadmeData struct {
	Name         string
	Region       string
	StateBucket  string
	Environments []EnvironmentData
	NATGateways  int
}

func RenderRoot(data RootData) (string, error) {
	return render("root.hcl.tmpl", data)
}

func RenderStack(data EnvironmentData) (string, error) {
	return render("stack.hcl.tmpl", data)
}

func RenderReadme(data ReadmeData) (string, error) {
	return render("readme.md.tmpl", data)
}

func render(name string, data any) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("render %s: %w", name, err)
	}
	return buf.String(), nil
}
