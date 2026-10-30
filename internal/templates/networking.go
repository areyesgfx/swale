package templates

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"
)

// NetworkingUnitHCL is the networking category's unit catalog entry
// (units/networking/terragrunt.hcl). It's identical for every generated
// project - the relative path back to the vendored module and the values.*
// references are fixed by the live/<account>/<env>/ layout, not by
// anything in a Spec - so unlike root.hcl or the stack file, it's embedded
// verbatim rather than run through text/template.
//
//go:embed networking_unit.hcl
var NetworkingUnitHCL string

//go:embed networking_stack.hcl.tmpl
var networkingStackSource string

var networkingStackTemplate = template.Must(template.New("networking_stack.hcl").Parse(networkingStackSource))

// NetworkingStackData is the input to the networking stack file template:
// the per-environment values that flow from Environment.Networking into
// the unit's values block.
type NetworkingStackData struct {
	VPCCIDR string
	NATMode string
}

// RenderNetworkingStack renders live/<account>/<env>/terragrunt.stack.hcl
// for the networking category.
func RenderNetworkingStack(data NetworkingStackData) (string, error) {
	var buf bytes.Buffer
	if err := networkingStackTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render networking stack: %w", err)
	}
	return buf.String(), nil
}
