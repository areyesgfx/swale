package cli

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/areyesgfx/swale/internal/generate"
	"github.com/areyesgfx/swale/internal/spec"
)

// initAnswers is what both "swale init" front doors collect: flags (for
// scripts and CI) and the interactive wizard. Each front door only fills
// this in; toSpec and spec.Validate own the rules, so the two can't drift.
type initAnswers struct {
	name            string
	region          string
	environments    int
	containerized   bool
	kind            string
	port            int
	healthCheckPath string
}

// toSpec builds the Spec without validating it, except for the name: the
// state bucket is named after the app, so a bad name would otherwise be
// reported twice. random supplies the bucket name's suffix.
func (a initAnswers) toSpec(random io.Reader) (spec.Spec, error) {
	if err := spec.CheckName(a.name); err != nil {
		return spec.Spec{}, fmt.Errorf("name: %w", err)
	}
	envs, err := spec.EnvironmentsFor(a.environments)
	if err != nil {
		return spec.Spec{}, err
	}
	bucket, err := spec.NewStateBucket(a.name, random)
	if err != nil {
		return spec.Spec{}, err
	}
	s := spec.Spec{Name: a.name, Region: a.region, Environments: envs, StateBucket: bucket}
	if a.containerized {
		s.App = spec.App{Containerized: true, Kind: spec.AppKind(a.kind)}
		if s.App.Kind == spec.AppKindWeb {
			s.App.Port = a.port
			s.App.HealthCheckPath = a.healthCheckPath
		}
	}
	return s, nil
}

// newInitCmd builds "swale init". random supplies the state bucket name's
// suffix: crypto/rand.Reader, or a fixed reader in tests.
func newInitCmd(random io.Reader) *cobra.Command {
	var a initAnswers

	cmd := &cobra.Command{
		Use:   "init [dir]",
		Short: "Create a new swale project",
		Long: `Create a new swale project in dir (default: the current directory): a
swale.yaml describing your app, and the Terragrunt/Terraform project
generated from it.

Run with no flags to answer a few questions about your app. Pass flags
instead to run non-interactively, e.g. in scripts or CI; any you leave out
take their defaults.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := projectDir(args)
			specPath := filepath.Join(dir, spec.FileName)
			// Checked before asking anything, so nobody answers every
			// question only to be told the project already exists.
			if _, err := os.Stat(specPath); err == nil {
				return fmt.Errorf("%s already exists; edit it and run \"swale generate\", or delete it to start over", specPath)
			} else if !errors.Is(err, fs.ErrNotExist) {
				return err
			}

			if a.name == "" {
				a.name = nameFromDir(dir)
			}
			if cmd.Flags().NFlag() == 0 {
				var err error
				if a, err = runWizard(a); err != nil {
					return err
				}
			} else if a.name == "" {
				return errors.New("--name is required, since the directory name isn't a valid app name")
			} else if !a.containerized {
				for _, flag := range []string{"kind", "port", "health-check-path"} {
					if cmd.Flags().Changed(flag) {
						return fmt.Errorf("--%s only applies to a containerized app, so it can't be combined with --containerized=false", flag)
					}
				}
			}

			// Render before writing anything, so a spec swale can't generate
			// leaves no half-created project behind.
			s, err := a.toSpec(random)
			if err != nil {
				return err
			}
			if err := s.Validate(); err != nil {
				return err
			}
			files, err := generate.Render(s)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
			if err := spec.Save(s, specPath); err != nil {
				return err
			}
			r, err := generate.Write(dir, files)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Created %s and generated your project:\n", specPath)
			printResult(out, r)
			printNextSteps(out, dir, s)
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&a.name, "name", "", "app name, used to name its AWS resources (default: the directory name)")
	flags.StringVar(&a.region, "region", "us-east-1", "AWS region to run in")
	flags.IntVar(&a.environments, "environments", 2, "how many environments: 1 (prod), 2 (stage, prod) or 3 (dev, stage, prod)")
	flags.BoolVar(&a.containerized, "containerized", true, "the app runs in a container (has a Dockerfile); false sets up only the network")
	flags.StringVar(&a.kind, "kind", string(spec.AppKindWeb), `what the app does: "web" (serves requests behind a load balancer) or "worker" (runs in the background)`)
	flags.IntVar(&a.port, "port", 8080, "port a web app listens on")
	flags.StringVar(&a.healthCheckPath, "health-check-path", "/", "path that answers with a 2xx or 3xx status when a web app is healthy")

	return cmd
}

// nameFromDir suggests the project directory's name as the app name, or
// returns "" when it isn't a valid one.
func nameFromDir(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	name := filepath.Base(abs)
	if spec.CheckName(name) != nil {
		return ""
	}
	return name
}
