// Package generate turns a spec.Spec into a real Terragrunt/Terraform
// project tree: root.hcl, a unit catalog + vendored module per category,
// and one live/<account>/<env>/ stack per environment.
package generate

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/areyesgfx/swale/internal/modules"
	"github.com/areyesgfx/swale/internal/spec"
	"github.com/areyesgfx/swale/internal/templates"
)

// Generate emits a swale project tree for s into outDir.
//
// v1 covers the networking category only, and only single-account specs -
// multi-account generation is a later phase.
//
// Per ADR 0003, Generate never overwrites an existing file: regenerating
// into a directory that already has swale-managed output is a conflict to
// report, not a refresh to apply silently.
func Generate(s spec.Spec, outDir string) error {
	if s.Accounts.Mode != spec.AccountModeSingle {
		return fmt.Errorf("generate: only single-account specs are supported (got mode %q); multi-account is a later phase", s.Accounts.Mode)
	}
	if len(s.Accounts.Accounts) < 1 {
		return errors.New("generate: spec has no accounts")
	}
	account := s.Accounts.Accounts[0]

	root, err := templates.RenderRoot(templates.RootData{
		StateBucket: fmt.Sprintf("swale-%s-tfstate", account.Name),
		Region:      s.Region,
	})
	if err != nil {
		return err
	}
	if err := writeNewFile(filepath.Join(outDir, "root.hcl"), root); err != nil {
		return err
	}

	unitPath := filepath.Join(outDir, "units", "networking", "terragrunt.hcl")
	if err := writeNewFile(unitPath, templates.NetworkingUnitHCL); err != nil {
		return err
	}

	for _, env := range s.Environments {
		stack, err := templates.RenderNetworkingStack(templates.NetworkingStackData{
			VPCCIDR: env.Networking.VPCCIDR,
			NATMode: string(env.Networking.NAT),
		})
		if err != nil {
			return err
		}
		stackPath := filepath.Join(outDir, "live", account.Name, env.Name, "terragrunt.stack.hcl")
		if err := writeNewFile(stackPath, stack); err != nil {
			return err
		}
	}

	if err := vendorModule(modules.Networking, "networking", filepath.Join(outDir, "modules", "networking")); err != nil {
		return err
	}

	return nil
}

// writeNewFile writes content to path, creating parent directories as
// needed. It refuses to overwrite an existing file: per ADR 0003, swale
// never rewrites generated output, so a pre-existing file at path is a
// conflict to report, not silently clobber.
func writeNewFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create directory for %q: %w", path, err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("generate: %q already exists; refusing to overwrite generated output (ADR 0003)", path)
		}
		return fmt.Errorf("create %q: %w", path, err)
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		return fmt.Errorf("write %q: %w", path, err)
	}
	return nil
}

// vendorModule copies the embedded module at srcDir (rooted inside src) to
// destDir, file by file, via writeNewFile - so vendoring gets the same
// no-overwrite conflict handling as every other generated file.
func vendorModule(src fs.FS, srcDir, destDir string) error {
	sub, err := fs.Sub(src, srcDir)
	if err != nil {
		return fmt.Errorf("vendor module %q: %w", srcDir, err)
	}
	return fs.WalkDir(sub, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		content, err := fs.ReadFile(sub, path)
		if err != nil {
			return fmt.Errorf("read vendored module file %q: %w", path, err)
		}
		return writeNewFile(filepath.Join(destDir, path), string(content))
	})
}
