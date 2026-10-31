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
		Name:         "acme-api",
		Region:       "us-east-1",
		Environments: []string{"stage", "prod"},
		StateBucket:  "acme-api-tfstate-0a1b2c3d",
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
		"README.md",
		"root.hcl",
		"units/networking/terragrunt.hcl",
		"units/networking/.terraform.lock.hcl",
		"live/stage/terragrunt.stack.hcl",
		"live/prod/terragrunt.stack.hcl",
		"modules/networking/main.tf",
		"modules/networking/outputs.tf",
		"modules/networking/variables.tf",
		"modules/networking/versions.tf",
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
	for _, want := range []string{`bucket  = "acme-api-tfstate-0a1b2c3d"`, `region  = "us-east-1"`, `App = "acme-api"`} {
		if !strings.Contains(root, want) {
			t.Errorf("root.hcl missing %q:\n%s", want, root)
		}
	}
}

func TestRenderStackContentPerEnv(t *testing.T) {
	got := render(t, sampleSpec())
	for path, wants := range map[string][]string{
		"live/stage/terragrunt.stack.hcl": {`name     = "acme-api-stage"`, `vpc_cidr = "10.20.0.0/16"`, `nat_mode = "single"`},
		"live/prod/terragrunt.stack.hcl":  {`name     = "acme-api-prod"`, `vpc_cidr = "10.30.0.0/16"`, `nat_mode = "per_az"`},
	} {
		for _, want := range wants {
			if !strings.Contains(got[path], want) {
				t.Errorf("%s missing %q:\n%s", path, want, got[path])
			}
		}
	}
}

// TestEveryKnownEnvironmentHasAProfile guards the link between spec and
// generate: spec.Validate accepts every name in spec.KnownEnvironments, so
// Render must know how to build each of them.
func TestEveryKnownEnvironmentHasAProfile(t *testing.T) {
	for _, env := range spec.KnownEnvironments {
		if _, ok := profiles[env]; !ok {
			t.Errorf("no profile for %q", env)
		}
	}
}

func TestProfilesDoNotOverlap(t *testing.T) {
	seen := map[string]string{}
	for env, p := range profiles {
		if other, dup := seen[p.vpcCIDR]; dup {
			t.Errorf("%s and %s share VPC CIDR %s", env, other, p.vpcCIDR)
		}
		seen[p.vpcCIDR] = env
	}
}

func TestRenderReadme(t *testing.T) {
	readme := render(t, sampleSpec())["README.md"]
	for _, want := range []string{
		"# acme-api",
		"- **stage**: a private network (`10.20.0.0/16`)",
		"Both zones share one NAT gateway",
		"- **prod**: a private network (`10.30.0.0/16`)",
		"Each zone has its own NAT gateway",
		"This project has 3 NAT gateways",
		"cd live/stage\n",
		"`acme-api-tfstate-0a1b2c3d`",
		"- `live/prod`",
	} {
		if !strings.Contains(readme, want) {
			t.Errorf("README.md missing %q:\n%s", want, readme)
		}
	}
}

func TestRenderReadmeSingleEnvironment(t *testing.T) {
	s := sampleSpec()
	s.Environments = []string{"prod"}
	readme := render(t, s)["README.md"]
	if strings.Contains(readme, "other environments") {
		t.Errorf("README.md for a single environment mentions other environments:\n%s", readme)
	}
	if !strings.Contains(readme, "This project has 2 NAT gateways") {
		t.Errorf("README.md for prod alone should count its 2 NAT gateways:\n%s", readme)
	}
}

func webSpec() spec.Spec {
	s := sampleSpec()
	s.App = spec.App{Containerized: true, Kind: spec.AppKindWeb, Port: 3000, HealthCheckPath: "/health"}
	return s
}

func TestRenderContainerizedApp(t *testing.T) {
	got := render(t, webSpec())
	for _, p := range []string{"units/service/terragrunt.hcl", "units/service/.terraform.lock.hcl", "modules/service/main.tf", "modules/service/variables.tf", "modules/service/outputs.tf", "modules/service/versions.tf"} {
		if _, ok := got[p]; !ok {
			t.Errorf("missing %q", p)
		}
	}
	for path, wants := range map[string][]string{
		"live/stage/terragrunt.stack.hcl": {`unit "service"`, `name          = "acme-api-stage"`, "desired_count = 1", "port              = 3000", `health_check_path = "/health"`},
		"live/prod/terragrunt.stack.hcl":  {"desired_count = 2"},
	} {
		for _, want := range wants {
			if !strings.Contains(got[path], want) {
				t.Errorf("%s missing %q:\n%s", path, want, got[path])
			}
		}
	}
	for _, want := range []string{"forwards them to your app on port 3000", "checks `/health`", "plain HTTP", "### Shipping your app", "--cluster acme-api-stage --service acme-api-stage", "Your app runs as 2 copies, spread across both zones."} {
		if !strings.Contains(got["README.md"], want) {
			t.Errorf("README.md missing %q", want)
		}
	}
}

func TestRenderWorker(t *testing.T) {
	s := sampleSpec()
	s.App = spec.App{Containerized: true, Kind: spec.AppKindWorker}
	got := render(t, s)
	stack := got["live/prod/terragrunt.stack.hcl"]
	if !strings.Contains(stack, `unit "service"`) {
		t.Errorf("worker stack has no service unit:\n%s", stack)
	}
	if strings.Contains(stack, "port") {
		t.Errorf("worker stack sets a port:\n%s", stack)
	}
	readme := got["README.md"]
	for _, want := range []string{"background worker", "### Shipping your app"} {
		if !strings.Contains(readme, want) {
			t.Errorf("README.md missing %q", want)
		}
	}
	if strings.Contains(readme, "load balancer serves") {
		t.Error("worker README talks about a load balancer")
	}
}

func TestRenderWithoutContainerOmitsService(t *testing.T) {
	got := render(t, sampleSpec())
	for p := range got {
		if strings.Contains(p, "service") {
			t.Errorf("rendered %q for an app that isn't containerized", p)
		}
	}
	if strings.Contains(got["live/prod/terragrunt.stack.hcl"], "service") {
		t.Error("stack mentions a service for an app that isn't containerized")
	}
	if !strings.Contains(got["README.md"], "## Adding your app later") {
		t.Error("README.md doesn't explain how to add the app later")
	}
}

// TestEveryUnitPinsItsProviders guards against a unit without a lock file,
// which would let its provider version drift between runs.
func TestEveryUnitPinsItsProviders(t *testing.T) {
	got := render(t, webSpec())
	for p := range got {
		if !strings.HasSuffix(p, "/terragrunt.hcl") || !strings.HasPrefix(p, "units/") {
			continue
		}
		lock := got[strings.TrimSuffix(p, "terragrunt.hcl")+".terraform.lock.hcl"]
		if !strings.Contains(lock, `provider "registry.terraform.io/hashicorp/aws"`) {
			t.Errorf("%s has no lock file pinning the AWS provider", p)
		}
	}
}

func TestRenderNoRemoteModulePinning(t *testing.T) {
	got := render(t, webSpec())
	for _, name := range []string{"networking", "service"} {
		path := "units/" + name + "/terragrunt.hcl"
		if strings.Contains(got[path], "ref=") {
			t.Errorf("%s references a remote module (ref=...), want a local vendored path:\n%s", path, got[path])
		}
		if !strings.Contains(got[path], "../../../../modules/"+name) {
			t.Errorf("%s does not reference the vendored module by local relative path:\n%s", path, got[path])
		}
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
