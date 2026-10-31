// Package templates holds the text/template sources swale renders into
// generated Terragrunt/Terraform projects (ADR 0001: text/template, not
// hclwrite, so generated files stay plain, readable, and hand-editable).
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
// depends on a Spec (its paths are fixed by the live/<account>/<env>/
// layout), so it's embedded verbatim instead of rendered.
//
//go:embed networking_unit.hcl
var NetworkingUnitHCL string

// RootData is the input to root.hcl. Terragrunt creates StateBucket on
// first use (ADR 0004), so it need not exist yet. Region applies to both
// the state backend and the provider block. Tags are merged into
// default_tags alongside the always-present ManagedBy=swale tag.
type RootData struct {
	StateBucket string
	Region      string
	Tags        map[string]string
}

// NetworkingStackData is the input to live/<account>/<env>/terragrunt.stack.hcl.
type NetworkingStackData struct {
	VPCCIDR string
	NATMode string
}

func RenderRoot(data RootData) (string, error) {
	return render("root.hcl.tmpl", data)
}

func RenderNetworkingStack(data NetworkingStackData) (string, error) {
	return render("networking_stack.hcl.tmpl", data)
}

func render(name string, data any) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("render %s: %w", name, err)
	}
	return buf.String(), nil
}
