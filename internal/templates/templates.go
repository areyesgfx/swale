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

// NetworkingUnitHCL and ServiceUnitHCL are units/<name>/terragrunt.hcl.
// Nothing in them depends on a Spec (their paths are fixed by the
// live/<env>/ layout), so they're embedded verbatim instead of rendered.
var (
	//go:embed networking_unit.hcl
	NetworkingUnitHCL string

	//go:embed service_unit.hcl
	ServiceUnitHCL string
)

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
// Service is nil when the app isn't containerized.
type EnvironmentData struct {
	Name         string
	ResourceName string
	VPCCIDR      string
	NATMode      string
	NATGateways  int
	Service      *ServiceData
}

// ServiceData is the containerized app's settings in one environment. Port
// is 0 for a background worker, which gets no load balancer.
type ServiceData struct {
	Port            int
	HealthCheckPath string
	DesiredCount    int
}

// ReadmeData is the input to the generated project's README.md.
// Environments are in the order the user listed them.
type ReadmeData struct {
	Name         string
	Region       string
	StateBucket  string
	Environments []EnvironmentData
	NATGateways  int
	App          AppData
}

// AppData is how the app runs, for the README. Kind is "" when the app
// isn't containerized, and Port is 0 unless Kind is "web".
type AppData struct {
	Kind            string
	Port            int
	HealthCheckPath string
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
