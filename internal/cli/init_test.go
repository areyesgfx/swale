package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"

	"github.com/areyesgfx/swale/internal/spec"
)

// fixedRandom stands in for crypto/rand.Reader, so state bucket names in
// tests are predictable: always "<name>-tfstate-0a1b2c3d".
func fixedRandom() io.Reader {
	return bytes.NewReader([]byte{0x0a, 0x1b, 0x2c, 0x3d})
}

// run executes cmd with args and returns what it printed to stdout.
func run(t *testing.T, cmd *cobra.Command, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

// runInit runs "swale init" with args in a fresh project directory, which
// it returns along with the command's output.
func runInit(t *testing.T, args ...string) (dir, out string, err error) {
	t.Helper()
	dir = t.TempDir()
	out, err = run(t, newInitCmd(fixedRandom()), append(args, dir)...)
	return dir, out, err
}

var validInitFlags = []string{"--name", "acme-api", "--region", "eu-west-1", "--environments", "3"}

func TestInitFlags(t *testing.T) {
	dir, out, err := runInit(t, validInitFlags...)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	got, err := spec.Load(filepath.Join(dir, spec.FileName))
	if err != nil {
		t.Fatalf("spec.Load: %v", err)
	}
	want := spec.Spec{
		Name:         "acme-api",
		Region:       "eu-west-1",
		Environments: []string{"dev", "stage", "prod"},
		App:          spec.App{Containerized: true, Kind: spec.AppKindWeb, Port: 8080, HealthCheckPath: "/"},
		StateBucket:  "acme-api-tfstate-0a1b2c3d",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("loaded spec = %#v, want %#v", got, want)
	}

	for _, p := range []string{"README.md", "live/dev/terragrunt.stack.hcl", "live/prod/terragrunt.stack.hcl", "units/service/terragrunt.hcl", "modules/service/main.tf"} {
		if _, err := os.Stat(filepath.Join(dir, p)); err != nil {
			t.Errorf("init did not generate %s: %v", p, err)
		}
	}
	for _, want := range []string{"created  root.hcl", "cd " + filepath.Join(dir, "live", "dev"), "terragrunt stack run plan --backend-bootstrap", "Shipping your app"} {
		if !strings.Contains(out, want) {
			t.Errorf("init output missing %q:\n%s", want, out)
		}
	}
}

func TestInitFlagDefaults(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "shop-web")
	if _, err := run(t, newInitCmd(fixedRandom()), "--region", "us-east-1", dir); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	got, err := spec.Load(filepath.Join(dir, spec.FileName))
	if err != nil {
		t.Fatalf("spec.Load: %v", err)
	}
	if got.Name != "shop-web" {
		t.Errorf("name = %q, want the directory name", got.Name)
	}
	if want := []string{"stage", "prod"}; !reflect.DeepEqual(got.Environments, want) {
		t.Errorf("environments = %v, want %v", got.Environments, want)
	}
}

func TestInitFlagsApp(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want spec.App
	}{
		{"web", []string{"--port", "3000", "--health-check-path", "/healthz"}, spec.App{Containerized: true, Kind: spec.AppKindWeb, Port: 3000, HealthCheckPath: "/healthz"}},
		{"worker ignores web defaults", []string{"--kind", "worker"}, spec.App{Containerized: true, Kind: spec.AppKindWorker}},
		{"worker with database", []string{"--kind", "worker", "--database", "postgres"}, spec.App{Containerized: true, Kind: spec.AppKindWorker, Database: spec.DatabasePostgres}},
		{"not containerized", []string{"--containerized=false"}, spec.App{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, _, err := runInit(t, append([]string{"--name", "acme-api"}, tt.args...)...)
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			got, err := spec.Load(filepath.Join(dir, spec.FileName))
			if err != nil {
				t.Fatalf("spec.Load: %v", err)
			}
			if got.App != tt.want {
				t.Errorf("app = %#v, want %#v", got.App, tt.want)
			}
			_, err = os.Stat(filepath.Join(dir, "units/service/terragrunt.hcl"))
			if exists := err == nil; exists != tt.want.Containerized {
				t.Errorf("units/service/terragrunt.hcl exists = %v, want %v", exists, tt.want.Containerized)
			}
		})
	}
}

func TestInitRefusesExistingSpec(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, spec.FileName)
	if err := os.WriteFile(specPath, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := run(t, newInitCmd(fixedRandom()), append(validInitFlags, dir)...)
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("Execute error = %v, want one saying the spec already exists", err)
	}
	if got, _ := os.ReadFile(specPath); string(got) != "mine" {
		t.Errorf("existing spec was overwritten with %q", got)
	}
}

func TestInitFlagErrors(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantSub string
	}{
		{"invalid name", []string{"--name", "Acme"}, "name: \"Acme\" must be lowercase"},
		{"invalid region", []string{"--name", "acme", "--region", "mars-1"}, "does not look like an AWS region"},
		{"too many environments", []string{"--name", "acme", "--environments", "4"}, "1 to 3 environments"},
		{"no environments", []string{"--name", "acme", "--environments", "0"}, "1 to 3 environments"},
		{"unknown kind", []string{"--name", "acme", "--kind", "cron"}, `"cron" is not a valid kind`},
		{"invalid port", []string{"--name", "acme", "--port", "0"}, "app.port: 0 is not a port number"},
		{"port without container", []string{"--name", "acme", "--containerized=false", "--port", "80"}, "--port only applies to a containerized app"},
		{"database without container", []string{"--name", "acme", "--containerized=false", "--database", "postgres"}, "--database only applies to a containerized app"},
		{"unknown database", []string{"--name", "acme", "--database", "mysql"}, `app.database: "mysql" is not supported`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, _, err := runInit(t, tt.args...)
			if err == nil {
				t.Fatalf("Execute: want error containing %q, got nil", tt.wantSub)
			}
			if !strings.Contains(err.Error(), tt.wantSub) {
				t.Errorf("Execute error = %q, want it to contain %q", err.Error(), tt.wantSub)
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 0 {
				t.Errorf("init left %d entries in the project directory despite the error", len(entries))
			}
		})
	}
}

func TestInitRequiresNameWhenDirNameIsInvalid(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Not_A_Name")
	_, err := run(t, newInitCmd(fixedRandom()), "--region", "us-east-1", dir)
	if err == nil || !strings.Contains(err.Error(), "--name is required") {
		t.Fatalf("Execute error = %v, want one asking for --name", err)
	}
}

// TestWizardForms drives the real huh forms through their accessible
// (line-based) mode, so the test needs no terminal. Each huh field reads
// stdin through its own buffered scanner, so input goes through
// OneByteReader to stop one field from swallowing the next field's line.
// Accessible mode doesn't offer pre-filled values as defaults, so every
// answer is typed.
func TestWizardForms(t *testing.T) {
	a := initAnswers{region: "us-east-1", environments: 2, containerized: true, kind: "web", database: noDatabase, port: 8080, healthCheckPath: "/"}
	answer(t, initForm(&a), "shop-web\n"+ // name
		"6\n"+ // region: Europe (Ireland)
		"1\n"+ // environments: prod only
		"1\n") // containerized: yes
	answer(t, appForm(&a), "1\n"+ // web
		"2\n") // database: PostgreSQL
	answer(t, webForm(&a), "3000\n"+ // port
		"/healthz\n") // health check path

	got, err := a.toSpec(fixedRandom())
	if err != nil {
		t.Fatalf("toSpec: %v", err)
	}
	want := spec.Spec{
		Name:         "shop-web",
		Region:       "eu-west-1",
		Environments: []string{"prod"},
		App:          spec.App{Containerized: true, Kind: spec.AppKindWeb, Database: spec.DatabasePostgres, Port: 3000, HealthCheckPath: "/healthz"},
		StateBucket:  "shop-web-tfstate-0a1b2c3d",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wizard spec = %#v, want %#v", got, want)
	}
}

func TestWizardSkipsQuestionsThatDoNotApply(t *testing.T) {
	worker := initAnswers{containerized: true, kind: "worker"}
	if webForm(&worker) != nil {
		t.Error("webForm asked a worker for its port")
	}
	none := initAnswers{containerized: false, kind: "web"}
	if appForm(&none) != nil || webForm(&none) != nil {
		t.Error("an app that isn't containerized was asked what it does")
	}
}

// answer runs form in accessible mode with input as the typed answers.
func answer(t *testing.T, form *huh.Form, input string) {
	t.Helper()
	err := form.
		WithAccessible(true).
		WithInput(iotest.OneByteReader(strings.NewReader(input))).
		WithOutput(io.Discard).
		Run()
	if err != nil {
		t.Fatalf("form: %v", err)
	}
}

func TestEveryWizardRegionIsValid(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range regions {
		if err := spec.CheckRegion(r.code); err != nil {
			t.Errorf("%s: %v", r.place, err)
		}
		if seen[r.code] {
			t.Errorf("%s is listed twice", r.code)
		}
		seen[r.code] = true
	}
}

func TestJoinAnd(t *testing.T) {
	for want, items := range map[string][]string{
		"prod":                {"prod"},
		"stage and prod":      {"stage", "prod"},
		"dev, stage and prod": {"dev", "stage", "prod"},
		"":                    nil,
	} {
		if got := joinAnd(items); got != want {
			t.Errorf("joinAnd(%v) = %q, want %q", items, got, want)
		}
	}
}

// TestBackableInput pins the huh v2.0.3 workaround: an invalid input must
// not block leaving the screen, but must still report its error once it
// has focus again.
func TestBackableInput(t *testing.T) {
	in := &backableInput{Input: huh.NewInput().Validate(spec.CheckName)}
	in.Focus()
	in.Blur()
	if err := in.Error(); err != nil {
		t.Errorf("Error() while leaving = %v, want nil", err)
	}
	in.Focus()
	if err := in.Error(); err == nil {
		t.Error("Error() after refocus = nil, want the validation error")
	}
}
