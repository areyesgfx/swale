package cli

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/areyesgfx/swale/internal/generate"
	"github.com/areyesgfx/swale/internal/spec"
)

func newGenerateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "generate [dir]",
		Short: "Generate the project described by swale.yaml",
		Long: `Generate the Terragrunt/Terraform project described by swale.yaml in dir
(default: the current directory).

swale never overwrites a file that already exists, since it may hold your
edits, so re-running is safe: after you change swale.yaml, for example to
add an environment, only the new files are created. To get swale's version
of an existing file, delete it and run generate again.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := projectDir(args)
			specPath := filepath.Join(dir, spec.FileName)
			s, err := spec.Load(specPath)
			if errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("no %s in %s; run \"swale init\" to create one", spec.FileName, dir)
			}
			if err != nil {
				return err
			}

			files, err := generate.Render(s)
			if err != nil {
				return err
			}
			r, err := generate.Write(dir, files)
			if err != nil {
				return err
			}
			printResult(cmd.OutOrStdout(), r)
			return nil
		},
	}
}

// projectDir is the project directory a command was given, defaulting to
// the current directory.
func projectDir(args []string) string {
	if len(args) == 1 {
		return args[0]
	}
	return "."
}

func printResult(w io.Writer, r generate.Result) {
	for _, p := range r.Created {
		fmt.Fprintf(w, "  created  %s\n", p)
	}
	for _, p := range r.Kept {
		fmt.Fprintf(w, "  kept     %s (differs from what swale.yaml generates)\n", p)
	}
	if len(r.Created) == 0 && len(r.Kept) == 0 {
		fmt.Fprintln(w, "Nothing to do: every file already matches swale.yaml.")
	}
	if len(r.Kept) > 0 {
		fmt.Fprintln(w, "\nswale never overwrites an existing file, since it may hold your edits.")
		fmt.Fprintln(w, "To get swale's version of a kept file, delete it and run swale generate again.")
	}
}

func printNextSteps(w io.Writer, dir string, s spec.Spec) {
	env := s.Environments[0]
	fmt.Fprintf(w, `
Next steps (%s explains each one):
  1. Sign in to AWS in this shell, for example with "aws sso login" or by setting AWS_PROFILE.
  2. Deploy the %s environment:
       cd %s
       terragrunt stack run plan --backend-bootstrap
       terragrunt stack run apply
`, filepath.Join(dir, "README.md"), env, filepath.Join(dir, filepath.FromSlash(generate.StackDir(env))))
	if s.App.Containerized {
		fmt.Fprintln(w, `  3. Push your app's image, as "Shipping your app" in the README shows.`)
	}
}
