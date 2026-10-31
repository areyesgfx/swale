package spec_test

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/areyesgfx/swale/internal/spec"
)

func valid() spec.Spec {
	return spec.Spec{
		Name:         "acme-api",
		Region:       "us-east-1",
		Environments: []string{"stage", "prod"},
		StateBucket:  "acme-api-tfstate-0a1b2c3d",
	}
}

func webApp() spec.App {
	return spec.App{Containerized: true, Kind: spec.AppKindWeb, Port: 8080, HealthCheckPath: "/health"}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	for name, app := range map[string]spec.App{
		"not containerized": {},
		"web":               webApp(),
		"worker":            {Containerized: true, Kind: spec.AppKindWorker},
	} {
		t.Run(name, func(t *testing.T) {
			want := valid()
			want.App = app
			path := filepath.Join(t.TempDir(), spec.FileName)
			if err := spec.Save(want, path); err != nil {
				t.Fatalf("Save: %v", err)
			}
			got, err := spec.Load(path)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("round-trip mismatch:\n got: %#v\nwant: %#v", got, want)
			}
		})
	}
}

// TestLoadWithoutApp keeps swale.yaml files written before the app
// section existed loadable: a missing app means not containerized.
func TestLoadWithoutApp(t *testing.T) {
	path := filepath.Join(t.TempDir(), spec.FileName)
	body := "name: acme-api\nregion: us-east-1\nenvironments: [prod]\nstate_bucket: acme-api-tfstate-0a1b2c3d\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := spec.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.App != (spec.App{}) {
		t.Errorf("app = %#v, want the zero value", got.App)
	}
}

func TestSaveWritesExplanatoryComments(t *testing.T) {
	path := filepath.Join(t.TempDir(), spec.FileName)
	if err := spec.Save(valid(), path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# swale.yaml describes your app.", "# The S3 bucket that stores Terraform state."} {
		if !strings.Contains(string(content), want) {
			t.Errorf("swale.yaml missing comment %q:\n%s", want, content)
		}
	}
}

func TestEnvironmentsFor(t *testing.T) {
	for count, want := range map[int][]string{
		1: {"prod"},
		2: {"stage", "prod"},
		3: {"dev", "stage", "prod"},
	} {
		got, err := spec.EnvironmentsFor(count)
		if err != nil {
			t.Fatalf("EnvironmentsFor(%d): %v", count, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("EnvironmentsFor(%d) = %v, want %v", count, got, want)
		}
	}
	for _, count := range []int{0, 4} {
		if _, err := spec.EnvironmentsFor(count); err == nil {
			t.Errorf("EnvironmentsFor(%d): want error, got nil", count)
		}
	}
}

func TestEnvironmentsForReturnsACopy(t *testing.T) {
	envs, err := spec.EnvironmentsFor(3)
	if err != nil {
		t.Fatal(err)
	}
	envs[0] = "changed"
	if spec.KnownEnvironments[0] != "dev" {
		t.Errorf("modifying EnvironmentsFor's result changed KnownEnvironments to %v", spec.KnownEnvironments)
	}
}

func TestNewStateBucket(t *testing.T) {
	got, err := spec.NewStateBucket("acme-api", bytes.NewReader([]byte{0x0a, 0x1b, 0x2c, 0x3d}))
	if err != nil {
		t.Fatalf("NewStateBucket: %v", err)
	}
	if want := "acme-api-tfstate-0a1b2c3d"; got != want {
		t.Errorf("NewStateBucket = %q, want %q", got, want)
	}
	if _, err := spec.NewStateBucket("acme-api", bytes.NewReader(nil)); err == nil {
		t.Error("NewStateBucket with no randomness available: want error, got nil")
	}
}

func TestValidateInvalid(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*spec.Spec)
		wantSub string
	}{
		{"empty name", func(s *spec.Spec) { s.Name = "" }, "name: must not be empty"},
		{"uppercase name", func(s *spec.Spec) { s.Name = "Acme" }, "lowercase letters"},
		{"name with double dash", func(s *spec.Spec) { s.Name = "acme--api" }, "single dashes"},
		{"name ending in dash", func(s *spec.Spec) { s.Name = "acme-" }, "single dashes"},
		{"name starting with digit", func(s *spec.Spec) { s.Name = "1acme" }, "starting with a letter"},
		{"name too long", func(s *spec.Spec) { s.Name = strings.Repeat("a", 21) }, "longer than 20"},
		{"empty region", func(s *spec.Spec) { s.Region = "" }, "region: must not be empty"},
		{"malformed region", func(s *spec.Spec) { s.Region = "not-a-region" }, "does not look like an AWS region"},
		{"no environments", func(s *spec.Spec) { s.Environments = nil }, "at least one environment"},
		{"unknown environment", func(s *spec.Spec) { s.Environments = []string{"qa"} }, `"qa" is not one of`},
		{"duplicate environment", func(s *spec.Spec) { s.Environments = []string{"prod", "prod"} }, "listed more than once"},
		{"empty state bucket", func(s *spec.Spec) { s.StateBucket = "" }, "state_bucket: must not be empty"},
		{"state bucket with dots", func(s *spec.Spec) { s.StateBucket = "acme.tfstate" }, "lowercase letters, numbers and dashes"},
		{"state bucket too long", func(s *spec.Spec) { s.StateBucket = strings.Repeat("a", 64) }, "3 to 63 characters"},
		{"kind without container", func(s *spec.Spec) { s.App.Kind = spec.AppKindWeb }, "only apply when containerized is true"},
		{"containerized without kind", func(s *spec.Spec) { s.App = spec.App{Containerized: true} }, "app.kind: must not be empty"},
		{"unknown kind", func(s *spec.Spec) { s.App = spec.App{Containerized: true, Kind: "cron"} }, `"cron" is not a valid kind`},
		{"web without port", func(s *spec.Spec) { s.App = webApp(); s.App.Port = 0 }, "app.port: 0 is not a port number"},
		{"web port too high", func(s *spec.Spec) { s.App = webApp(); s.App.Port = 70000 }, "app.port: 70000"},
		{"health check without slash", func(s *spec.Spec) { s.App = webApp(); s.App.HealthCheckPath = "health" }, "app.health_check_path"},
		{"health check with interpolation", func(s *spec.Spec) { s.App = webApp(); s.App.HealthCheckPath = "/${evil}" }, "app.health_check_path"},
		{"health check with quote", func(s *spec.Spec) { s.App = webApp(); s.App.HealthCheckPath = `/a"b` }, "app.health_check_path"},
		{"worker with port", func(s *spec.Spec) { s.App = spec.App{Containerized: true, Kind: spec.AppKindWorker, Port: 80} }, "only apply to web apps"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := valid()
			tt.mutate(&s)
			err := s.Validate()
			if err == nil {
				t.Fatalf("Validate: want error containing %q, got nil", tt.wantSub)
			}
			if !strings.Contains(err.Error(), tt.wantSub) {
				t.Errorf("Validate error = %q, want it to contain %q", err.Error(), tt.wantSub)
			}
		})
	}
}

func TestValidateReportsAllProblems(t *testing.T) {
	s := valid()
	s.Name = ""
	s.Environments = []string{"qa"}
	err := s.Validate()
	if err == nil {
		t.Fatal("Validate: want error, got nil")
	}
	for _, want := range []string{"name: must not be empty", `"qa" is not one of`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Validate error = %q, want it to contain %q", err.Error(), want)
		}
	}
}

func TestSaveRefusesInvalidSpec(t *testing.T) {
	s := valid()
	s.Region = "nope"
	path := filepath.Join(t.TempDir(), spec.FileName)
	if err := spec.Save(s, path); err == nil {
		t.Fatal("Save: want error for invalid spec, got nil")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("Save wrote a file for an invalid spec (stat err = %v)", err)
	}
}

func TestLoadRejectsUnknownField(t *testing.T) {
	path := filepath.Join(t.TempDir(), spec.FileName)
	body := "name: acme-api\ntypo_field: oops\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, err := spec.Load(path)
	if err == nil {
		t.Fatal("Load: want error for unknown field, got nil")
	}
	if !strings.Contains(err.Error(), "typo_field") {
		t.Errorf("Load error = %q, want it to mention the unknown field", err.Error())
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := spec.Load(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err == nil {
		t.Fatal("Load: want error for missing file, got nil")
	}
}
