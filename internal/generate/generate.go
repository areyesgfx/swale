// Package generate turns a spec.Spec into a Terragrunt/Terraform project:
// root.hcl, a unit catalog and vendored module per component, and one
// live/<env>/ stack per environment.
//
// Render decides every file's path and content without touching disk;
// Write is the only part that does I/O. Keeping the rules in a pure
// function means a spec swale can't generate fails before anything is
// written, and tests can check output without a filesystem.
package generate

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/areyesgfx/swale/internal/modules"
	"github.com/areyesgfx/swale/internal/spec"
	"github.com/areyesgfx/swale/internal/templates"
)

// File is one file of a generated project. Path is slash-separated and
// relative to the project root.
type File struct {
	Path    string
	Content string
}

// profile is the infrastructure swale derives for an environment, so the
// user never has to choose network ranges or NAT layouts.
//
// Every environment has its own fixed address range, rather than one
// assigned by its position in swale.yaml, so adding or removing an
// environment never shifts another's, and ranges never overlap in case the
// networks are ever connected. Only prod pays for a NAT gateway per
// availability zone.
type profile struct {
	vpcCIDR string
	natMode string
}

var profiles = map[string]profile{
	"dev":   {vpcCIDR: "10.10.0.0/16", natMode: "single"},
	"stage": {vpcCIDR: "10.20.0.0/16", natMode: "single"},
	"prod":  {vpcCIDR: "10.30.0.0/16", natMode: "per_az"},
}

// azCount matches the networking module's az_count default.
const azCount = 2

// Render returns every file of the project generated from s, in a stable
// order. s must be valid (see spec.Spec.Validate).
func Render(s spec.Spec) ([]File, error) {
	envs := make([]templates.EnvironmentData, 0, len(s.Environments))
	natGateways := 0
	for _, name := range s.Environments {
		p, ok := profiles[name]
		if !ok {
			return nil, fmt.Errorf("generate: no infrastructure profile for environment %q", name)
		}
		env := templates.EnvironmentData{
			Name:         name,
			ResourceName: s.Name + "-" + name,
			VPCCIDR:      p.vpcCIDR,
			NATMode:      p.natMode,
			NATGateways:  1,
		}
		if p.natMode == "per_az" {
			env.NATGateways = azCount
		}
		natGateways += env.NATGateways
		envs = append(envs, env)
	}

	root, err := templates.RenderRoot(templates.RootData{
		StateBucket: s.StateBucket,
		Region:      s.Region,
		Tags:        map[string]string{"App": s.Name},
	})
	if err != nil {
		return nil, err
	}
	readme, err := templates.RenderReadme(templates.ReadmeData{
		Name:         s.Name,
		Region:       s.Region,
		StateBucket:  s.StateBucket,
		Environments: envs,
		NATGateways:  natGateways,
	})
	if err != nil {
		return nil, err
	}
	files := []File{
		{Path: ".gitignore", Content: templates.GitIgnore},
		{Path: "README.md", Content: readme},
		{Path: "root.hcl", Content: root},
		{Path: "units/networking/terragrunt.hcl", Content: templates.NetworkingUnitHCL},
	}

	for _, env := range envs {
		stack, err := templates.RenderStack(env)
		if err != nil {
			return nil, err
		}
		files = append(files, File{Path: path.Join(StackDir(env.Name), "terragrunt.stack.hcl"), Content: stack})
	}

	networking, err := vendorModule(modules.Networking, "networking")
	if err != nil {
		return nil, err
	}
	return append(files, networking...), nil
}

// StackDir is the slash-separated directory, relative to the project root,
// that holds env's terragrunt.stack.hcl.
func StackDir(env string) string {
	return path.Join("live", env)
}

// vendorModule returns the embedded module rooted at name inside src as
// files under modules/<name>/, so every generated project carries its own
// copy of the module source instead of depending on a remote module repo.
func vendorModule(src fs.FS, name string) ([]File, error) {
	var files []File
	err := fs.WalkDir(src, name, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		content, err := fs.ReadFile(src, p)
		if err != nil {
			return err
		}
		files = append(files, File{Path: path.Join("modules", p), Content: string(content)})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("vendor module %q: %w", name, err)
	}
	return files, nil
}

// Result reports, by path, what Write did with each file. Kept holds files
// that already existed with different content.
type Result struct {
	Created   []string
	Unchanged []string
	Kept      []string
}

// Write writes files under dir, creating directories as needed. It never
// modifies a file that already exists, because once a project is generated
// its files belong to the user and may hold their edits. That makes Write
// safe to re-run: after adding an environment to swale.yaml, only the new
// environment's files are created.
func Write(dir string, files []File) (Result, error) {
	var r Result
	for _, f := range files {
		p := filepath.Join(dir, filepath.FromSlash(f.Path))
		existing, err := os.ReadFile(p)
		switch {
		case err == nil && string(existing) == f.Content:
			r.Unchanged = append(r.Unchanged, f.Path)
			continue
		case err == nil:
			r.Kept = append(r.Kept, f.Path)
			continue
		case !errors.Is(err, fs.ErrNotExist):
			return r, fmt.Errorf("read %q: %w", p, err)
		}
		if err := writeNewFile(p, f.Content); err != nil {
			return r, err
		}
		r.Created = append(r.Created, f.Path)
	}
	return r, nil
}

// writeNewFile creates path with content. O_EXCL makes the create fail if
// the file appeared after Write checked for it, so even that race can't
// overwrite a user's file.
func writeNewFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create directory for %q: %w", path, err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("create %q: %w", path, err)
	}
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		return fmt.Errorf("write %q: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("write %q: %w", path, err)
	}
	return nil
}
