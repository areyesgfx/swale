package templates

import (
	"strings"
	"testing"
)

func TestRenderRoot(t *testing.T) {
	out, err := RenderRoot(RootData{
		StateBucket: "acme-swale-tfstate",
		Region:      "us-east-1",
		Tags:        map[string]string{"Environment": "dev", "Account": "workloads"},
	})
	if err != nil {
		t.Fatalf("RenderRoot: %v", err)
	}

	for _, want := range []string{
		`bucket  = "acme-swale-tfstate"`,
		`region  = "us-east-1"`,
		`region = "us-east-1"`,
		"use_lockfile = true",
		`ManagedBy = "swale"`,
		// Sorted by key (text/template's guarantee for map iteration), not
		// insertion order, so output is deterministic across runs.
		"Account = \"workloads\"",
		"Environment = \"dev\"",
		// The literal HCL that strips the stacks materialization segment
		// from the state key (docs/research/terragrunt-stacks.md, finding 2).
		// Verified against a real
		// `terragrunt stack generate` + `terragrunt render` run that this
		// actually produces "live/dev/hello" from
		// "live/dev/.terragrunt-stack/hello".
		`replace(path_relative_to_include(), ".terragrunt-stack/", "")`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("RenderRoot output missing %q\ngot:\n%s", want, out)
		}
	}

	if i := strings.Index(out, "Account"); i >= 0 {
		if j := strings.Index(out, "Environment"); j >= 0 && j < i {
			t.Error("Tags rendered out of sorted order: Environment appeared before Account")
		}
	}

	if strings.Contains(out, "{{") || strings.Contains(out, "}}") {
		t.Errorf("RenderRoot output has unexecuted template directives:\n%s", out)
	}
}

func TestRenderRootNoExtraTags(t *testing.T) {
	out, err := RenderRoot(RootData{StateBucket: "b", Region: "us-east-1"})
	if err != nil {
		t.Fatalf("RenderRoot: %v", err)
	}
	if !strings.Contains(out, `ManagedBy = "swale"`) {
		t.Errorf("RenderRoot output missing default ManagedBy tag:\n%s", out)
	}
	// With no extra tags, the tags block should hold exactly the default
	// tag and nothing else - no leftover blank lines or stray commas from
	// an empty range over .Tags.
	start := strings.Index(out, "tags = {")
	end := strings.Index(out, "}\n      }\n    }")
	if start == -1 || end == -1 || end < start {
		t.Fatalf("could not locate tags block in output:\n%s", out)
	}
	body := out[start+len("tags = {") : end+1]
	if strings.Count(body, "=") != 1 {
		t.Errorf("tags block body = %q, want exactly one tag (ManagedBy)", body)
	}
}

func TestRenderRootVersionConstraint(t *testing.T) {
	out, err := RenderRoot(RootData{StateBucket: "b", Region: "us-east-1"})
	if err != nil {
		t.Fatalf("RenderRoot: %v", err)
	}
	if !strings.Contains(out, `terragrunt_version_constraint = ">= 1.1.0"`) {
		t.Errorf("RenderRoot output missing terragrunt_version_constraint:\n%s", out)
	}
}
