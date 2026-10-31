package generate

import (
	"os"
	"path/filepath"
	"reflect"
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

// render calls Render on s and returns its files keyed by path.
func render(t *testing.T, s spec.Spec) map[string]string {
	t.Helper()
	files, err := Render(s)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	byPath := make(map[string]string, len(files))
	for _, f := range files {
		if _, dup := byPath[f.Path]; dup {
			t.Errorf("Render returned %q twice", f.Path)
		}
		byPath[f.Path] = f.Content
	}
	return byPath
}

func TestRenderProducesExpectedTree(t *testing.T) {
	got := render(t, sampleSpec())
	want := []string{
		".gitignore",
		"root.hcl",
		"units/networking/terragrunt.hcl",
		"live/main/dev/terragrunt.stack.hcl",
		"live/main/prod/terragrunt.stack.hcl",
		"modules/networking/main.tf",
		"modules/networking/outputs.tf",
		"modules/networking/variables.tf",
	}
	for _, p := range want {
		if _, ok := got[p]; !ok {
			t.Errorf("missing %q", p)
		}
	}
	if len(got) != len(want) {
		t.Errorf("Render returned %d files, want %d", len(got), len(want))
	}
}

func TestRenderIsDeterministic(t *testing.T) {
	first, err := Render(sampleSpec())
	if err != nil {
		t.Fatal(err)
	}
	second, err := Render(sampleSpec())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Error("two Render calls on the same spec returned different files")
	}
}

func TestRenderRootHCLContent(t *testing.T) {
	root := render(t, sampleSpec())["root.hcl"]
	for _, want := range []string{`bucket  = "swale-main-tfstate"`, `region  = "us-east-1"`} {
		if !strings.Contains(root, want) {
			t.Errorf("root.hcl missing %q:\n%s", want, root)
		}
	}
}

func TestRenderStackContentPerEnv(t *testing.T) {
	got := render(t, sampleSpec())
	for path, wants := range map[string][]string{
		"live/main/dev/terragrunt.stack.hcl":  {`vpc_cidr = "10.0.0.0/16"`, `nat_mode = "single"`},
		"live/main/prod/terragrunt.stack.hcl": {`vpc_cidr = "10.1.0.0/16"`, `nat_mode = "per_az"`},
	} {
		for _, want := range wants {
			if !strings.Contains(got[path], want) {
				t.Errorf("%s missing %q:\n%s", path, want, got[path])
			}
		}
	}
}

func TestRenderRejectsMultiAccount(t *testing.T) {
	s := sampleSpec()
	s.Accounts = spec.AccountsSpec{
		Mode: spec.AccountModeMulti,
		Accounts: []spec.Account{
			{Name: "mgmt", IsManagement: true},
			{Name: "dev"},
		},
	}
	_, err := Render(s)
	if err == nil {
		t.Fatal("Render: want error for multi-account spec, got nil")
	}
	if !strings.Contains(err.Error(), "single-account") {
		t.Errorf("Render error = %q, want it to mention single-account", err.Error())
	}
}

func TestRenderNoRemoteModulePinning(t *testing.T) {
	unit := render(t, sampleSpec())["units/networking/terragrunt.hcl"]
	if strings.Contains(unit, "ref=") {
		t.Errorf("units/networking/terragrunt.hcl references a remote module (ref=...), want a local vendored path:\n%s", unit)
	}
	if !strings.Contains(unit, "../../../../../modules/networking") {
		t.Errorf("units/networking/terragrunt.hcl does not reference the vendored module by local relative path:\n%s", unit)
	}
}

func TestWriteCreatesFiles(t *testing.T) {
	dir := t.TempDir()
	files := []File{{Path: "a.txt", Content: "a"}, {Path: "nested/b.txt", Content: "b"}}

	r, err := Write(dir, files)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if want := (Result{Created: []string{"a.txt", "nested/b.txt"}}); !reflect.DeepEqual(r, want) {
		t.Errorf("Write result = %+v, want %+v", r, want)
	}
	for _, f := range files {
		got, err := os.ReadFile(filepath.Join(dir, f.Path))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != f.Content {
			t.Errorf("%s = %q, want %q", f.Path, got, f.Content)
		}
	}
}

func TestWriteNeverOverwrites(t *testing.T) {
	dir := t.TempDir()
	if _, err := Write(dir, []File{{Path: "same.txt", Content: "same"}, {Path: "edited.txt", Content: "original"}}); err != nil {
		t.Fatal(err)
	}
	edited := filepath.Join(dir, "edited.txt")
	if err := os.WriteFile(edited, []byte("the user's edit"), 0o644); err != nil {
		t.Fatal(err)
	}

	r, err := Write(dir, []File{
		{Path: "same.txt", Content: "same"},
		{Path: "edited.txt", Content: "original"},
		{Path: "new.txt", Content: "new"},
	})
	if err != nil {
		t.Fatalf("second Write: %v", err)
	}
	want := Result{Created: []string{"new.txt"}, Unchanged: []string{"same.txt"}, Kept: []string{"edited.txt"}}
	if !reflect.DeepEqual(r, want) {
		t.Errorf("second Write result = %+v, want %+v", r, want)
	}
	got, err := os.ReadFile(edited)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "the user's edit" {
		t.Errorf("edited.txt = %q, want the user's edit kept", got)
	}
}
