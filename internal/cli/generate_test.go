package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateRecreatesMissingFilesAndKeepsEdits(t *testing.T) {
	dir, _, err := runInit(t, validInitFlags...)
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	prodStack := filepath.Join(dir, "live/main/prod/terragrunt.stack.hcl")
	if err := os.Remove(prodStack); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(dir, "root.hcl")
	if err := os.WriteFile(root, []byte("# edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := run(t, newGenerateCmd(), dir)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	for _, want := range []string{"created  live/main/prod/terragrunt.stack.hcl", "kept     root.hcl"} {
		if !strings.Contains(out, want) {
			t.Errorf("generate output missing %q:\n%s", want, out)
		}
	}
	if _, err := os.Stat(prodStack); err != nil {
		t.Errorf("deleted stack was not recreated: %v", err)
	}
	if got, _ := os.ReadFile(root); string(got) != "# edited\n" {
		t.Errorf("root.hcl = %q, want the edit kept", got)
	}
}

func TestGenerateNothingToDo(t *testing.T) {
	dir, _, err := runInit(t, validInitFlags...)
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	out, err := run(t, newGenerateCmd(), dir)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if !strings.Contains(out, "Nothing to do") {
		t.Errorf("generate output = %q, want it to say there's nothing to do", out)
	}
}

func TestGenerateWithoutSpec(t *testing.T) {
	_, err := run(t, newGenerateCmd(), t.TempDir())
	if err == nil || !strings.Contains(err.Error(), `run "swale init"`) {
		t.Fatalf("generate error = %v, want one pointing at swale init", err)
	}
}
