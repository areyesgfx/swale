# Terragrunt stacks findings

These findings come from `stacks-lab/`, a throwaway prototype built for SWA-5 to answer questions about Terragrunt stacks materialization before the real `root.hcl` template ([ADR 0005](../adr/0005-terragrunt-stacks.md)) was written.
The prototype was removed in SWA-15 once the real template in `internal/templates/` and the CI-validated output in `examples/` replaced it.
Its source is still in git history at commit `2259857`.

The prototype had this layout, which the findings below refer to:

- `root.hcl`: generates a `local` backend that writes state to `stacks-lab/.tfstate/${path_relative_to_include()}/terraform.tfstate`, plus a `DEBUG_relative_path.txt` file containing the raw `path_relative_to_include()` value.
- `units/hello/terragrunt.hcl`: includes `root.hcl` via `find_in_parent_folders`, sources `modules/hello`, and passes `inputs = { name = values.env }`.
- `modules/hello/main.tf`: a single `terraform_data` resource whose input is `"hello, ${var.name}"`, exposed as the `greeting` output.
- `live/dev/terragrunt.stack.hcl`: declares one `unit "hello"` with `path = "hello"` and `values = { env = "dev" }`.

It was run with:

```
cd live/dev
terragrunt stack generate
terragrunt stack run apply --non-interactive
```

Terragrunt version used: `1.1.0` (installed via Homebrew, 2026-07-03).

## 1. Does `find_in_parent_folders("root.hcl")` resolve from inside the materialized `.terragrunt-stack/` unit?

Yes.
`units/hello/terragrunt.hcl` includes root via `find_in_parent_folders("root.hcl")`, and that file gets copied verbatim into `live/dev/.terragrunt-stack/hello/terragrunt.hcl` by `terragrunt stack generate`.
`stack run apply` resolved the include and ran cleanly, so the parent-folder search walks up through `.terragrunt-stack/` the same way it would through any other directory nesting - it found `stacks-lab/root.hcl`.

## 2. What does `path_relative_to_include()` return - does `.terragrunt-stack` appear in the path?

Yes, it appears.
Confirmed by dumping `local.relative_path` (= `path_relative_to_include()`) to a generated debug file from `root.hcl`.
The observed value was:

```
live/dev/.terragrunt-stack/hello
```

This also showed up in the generated backend config's state path: `.../stacks-lab/.tfstate/live/dev/.terragrunt-stack/hello/terraform.tfstate`.

**Consequence for the real `root.hcl` template:** a naive state-key derivation of `path_relative_to_include()` will bake `.terragrunt-stack/<unit-path>` into every state key.
That's noise that has nothing to do with the logical account/env/component identity - it's an artifact of how stacks materializes units.
The template needs to either strip a literal `.terragrunt-stack/` segment out of the derived path with an HCL string function, or derive the state key from something stacks-aware instead (e.g. the unit's declared `path` attribute in `terragrunt.stack.hcl`, which stays a clean logical name regardless of materialization depth).

## 3. How does a value flow from the stack file to the module input?

`terragrunt.stack.hcl`'s `unit` block has a `values = { env = "dev" }` map.
When `stack generate` runs, Terragrunt writes a sibling file next to the copied unit config: `.terragrunt-stack/hello/terragrunt.values.hcl`, containing just `env = "dev"`.
The copied `terragrunt.hcl` then reads it back through an implicit `values` object (`values.env`), which gets passed to the module as a normal Terraform input (`inputs = { name = values.env }`).
Apply output confirmed the full chain worked: `greeting = "hello, dev"`.

Where per-env knobs live now: not in a classic `env.hcl` read via `find_in_parent_folders` - they live directly in the `values` block of the unit declaration inside `terragrunt.stack.hcl`.
That's the natural home for per-env overrides in the stacks model; a project-wide `env.hcl` pattern doesn't fit as cleanly since each unit's values are declared where the unit itself is declared.

## 4. What happens to manual edits in `.terragrunt-stack/` on re-running `stack generate`?

They're wiped.
Appended a comment line directly to `.terragrunt-stack/hello/terragrunt.hcl`, re-ran `terragrunt stack generate`, and the file was fully regenerated from the source unit - the manual edit was gone.
Confirms `.terragrunt-stack/` is a disposable build artifact, matching ADR 0005's call to gitignore it.

## 5. Which Terragrunt version, and how will generated projects pin it?

`1.1.0`, installed via `brew install terragrunt` (not previously present on this machine).
Terragrunt supports a `terragrunt_version_constraint` attribute at the top of a `terragrunt.hcl` file (e.g. `terragrunt_version_constraint = ">= 1.1.0"`), which fails the run if the installed binary doesn't satisfy it.
The generated `root.hcl` template should set this so every generated project pins/enforces a minimum Terragrunt version at generation time, rather than relying on whatever version happens to be on a developer's PATH.
Not independently re-verified against a mismatched version in this session - flagging that as an assumption carried over from Terragrunt's documented behavior, worth a quick sanity check before it's load-bearing in the real template.
