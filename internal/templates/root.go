// Package templates holds the text/template sources swale renders into
// generated Terragrunt/Terraform projects (ADR 0001: text/template, not
// hclwrite, so generated files stay plain, readable, and hand-editable).
package templates

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"
)

//go:embed root.hcl.tmpl
var rootHCLSource string

var rootHCLTemplate = template.Must(template.New("root.hcl").Parse(rootHCLSource))

// RootData is the input to the root.hcl template: the per-project values
// that can't be baked into the template itself (state backend location and
// the tags every AWS resource in the project should carry by default).
type RootData struct {
	// StateBucket is the S3 bucket Terragrunt stores state in. Terragrunt
	// creates it on first use (ADR 0004), so it does not need to exist yet.
	StateBucket string
	// Region is the AWS region for both the state backend and the
	// generated provider block.
	Region string
	// Tags are merged into default_tags alongside the always-present
	// ManagedBy=swale tag. Rendered in sorted key order for stable output.
	Tags map[string]string
}

// RenderRoot renders root.hcl for the given data.
func RenderRoot(data RootData) (string, error) {
	var buf bytes.Buffer
	if err := rootHCLTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render root.hcl: %w", err)
	}
	return buf.String(), nil
}
