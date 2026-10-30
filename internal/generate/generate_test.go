package generate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/areyesgfx/swale/internal/spec"
)

func sampleSpec() spec.Spec {
	return spec.Spec{
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
}

func TestGenerateProducesExpectedTree(t *testing.T) {
	out := t.TempDir()
	if err := Generate(sampleSpec(), out); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	wantFiles := []string{
		"root.hcl",
		"units/networking/terragrunt.hcl",
		"live/main/dev/terragrunt.stack.hcl",
		"live/main/prod/terragrunt.stack.hcl",
		"modules/networking/main.tf",
		"modules/networking/variables.tf",
		"modules/networking/outputs.tf",
	}
	for _, rel := range wantFiles {
		if _, err := os.Stat(filepath.Join(out, rel)); err != nil {
			t.Errorf("expected file %q: %v", rel, err)
		}
	}
}

func TestGenerateRootHCLContent(t *testing.T) {
	out := t.TempDir()
	if err := Generate(sampleSpec(), out); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(out, "root.hcl"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`bucket  = "swale-main-tfstate"`, `region  = "us-east-1"`} {
		if !strings.Contains(string(content), want) {
			t.Errorf("root.hcl missing %q:\n%s", want, content)
		}
	}
}

func TestGenerateStackContentPerEnv(t *testing.T) {
	out := t.TempDir()
	if err := Generate(sampleSpec(), out); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	dev, err := os.ReadFile(filepath.Join(out, "live/main/dev/terragrunt.stack.hcl"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`vpc_cidr = "10.0.0.0/16"`, `nat_mode = "single"`} {
		if !strings.Contains(string(dev), want) {
			t.Errorf("dev stack missing %q:\n%s", want, dev)
		}
	}

	prod, err := os.ReadFile(filepath.Join(out, "live/main/prod/terragrunt.stack.hcl"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`vpc_cidr = "10.1.0.0/16"`, `nat_mode = "per_az"`} {
		if !strings.Contains(string(prod), want) {
			t.Errorf("prod stack missing %q:\n%s", want, prod)
		}
	}
}

func TestGenerateRejectsMultiAccount(t *testing.T) {
	s := sampleSpec()
	s.Accounts = spec.AccountsSpec{
		Mode: spec.AccountModeMulti,
		Accounts: []spec.Account{
			{Name: "mgmt", IsManagement: true},
			{Name: "dev"},
		},
	}
	err := Generate(s, t.TempDir())
	if err == nil {
		t.Fatal("Generate: want error for multi-account spec, got nil")
	}
	if !strings.Contains(err.Error(), "single-account") {
		t.Errorf("Generate error = %q, want it to mention single-account", err.Error())
	}
}

func TestGenerateRefusesToOverwrite(t *testing.T) {
	out := t.TempDir()
	if err := Generate(sampleSpec(), out); err != nil {
		t.Fatalf("first Generate: %v", err)
	}
	err := Generate(sampleSpec(), out)
	if err == nil {
		t.Fatal("second Generate: want error for pre-existing output, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("second Generate error = %q, want it to mention the file already existing", err.Error())
	}
}

func TestGenerateNoRemoteModulePinning(t *testing.T) {
	out := t.TempDir()
	if err := Generate(sampleSpec(), out); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(out, "units/networking/terragrunt.hcl"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "ref=") {
		t.Errorf("units/networking/terragrunt.hcl references a remote module (ref=...), want a local vendored path:\n%s", content)
	}
	if !strings.Contains(string(content), "../../../../../modules/networking") {
		t.Errorf("units/networking/terragrunt.hcl does not reference the vendored module by local relative path:\n%s", content)
	}
}
