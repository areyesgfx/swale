package generate

import (
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/areyesgfx/swale/internal/spec"
)

var update = flag.Bool("update", false, "regenerate examples/ from their swale.yaml instead of diffing against them")

// TestExamplesGolden regenerates every examples/<name>/ project from its
// swale.yaml and fails on any difference from what's committed: a changed,
// missing, or extra file. After an intentional template or module change,
// run `go test ./internal/generate -update` and review the examples/ diff
// like any other code change.
func TestExamplesGolden(t *testing.T) {
	specs, err := filepath.Glob(filepath.Join("..", "..", "examples", "*", "swale.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) == 0 {
		t.Fatal("no examples/*/swale.yaml found")
	}

	for _, specPath := range specs {
		dir := filepath.Dir(specPath)
		t.Run(filepath.Base(dir), func(t *testing.T) {
			s, err := spec.Load(specPath)
			if err != nil {
				t.Fatal(err)
			}

			if *update {
				if err := removeAllExcept(dir, "swale.yaml"); err != nil {
					t.Fatal(err)
				}
				if err := Generate(s, dir); err != nil {
					t.Fatal(err)
				}
				return
			}

			out := t.TempDir()
			if err := Generate(s, out); err != nil {
				t.Fatal(err)
			}
			got := readTree(t, out)
			want := readTree(t, dir)
			delete(want, "swale.yaml")

			for path, content := range got {
				wantContent, ok := want[path]
				switch {
				case !ok:
					t.Errorf("%s: generated but not committed", path)
				case content != wantContent:
					t.Errorf("%s: generated output differs from the committed file", path)
				}
			}
			for path := range want {
				if _, ok := got[path]; !ok {
					t.Errorf("%s: committed but no longer generated", path)
				}
			}
			if t.Failed() {
				t.Log("if this change is intended, run: go test ./internal/generate -update")
			}
		})
	}
}

// readTree returns every file under root, keyed by slash-separated path
// relative to root.
func readTree(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = string(content)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func removeAllExcept(dir, keep string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.Name() == keep {
			continue
		}
		if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}
