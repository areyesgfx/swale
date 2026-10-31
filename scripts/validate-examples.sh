#!/usr/bin/env bash
# Checks that every generated project under examples/ is formatted and
# passes `terraform validate` through Terragrunt. CI runs this; run it
# locally the same way: ./scripts/validate-examples.sh
#
# It never touches AWS: credentials are scrubbed from the environment and
# Terraform is initialized with -backend=false, so the S3 state backend in
# root.hcl is never contacted (and its bucket is never auto-created).
# Each example is validated in a temp copy so Terragrunt's build artifacts
# (.terragrunt-stack/, .terragrunt-cache/) never land in examples/, where
# they would fail the golden-file test.
set -euo pipefail

unset AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY AWS_SESSION_TOKEN AWS_PROFILE AWS_DEFAULT_PROFILE
export AWS_CONFIG_FILE=/dev/null
export AWS_SHARED_CREDENTIALS_FILE=/dev/null
export AWS_EC2_METADATA_DISABLED=true
export TG_NON_INTERACTIVE=true

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

shopt -s nullglob
examples=("$repo_root"/examples/*/)
if [ ${#examples[@]} -eq 0 ]; then
  echo "no examples found under examples/" >&2
  exit 1
fi

for example in "${examples[@]}"; do
  name="$(basename "$example")"
  echo "==> $name"
  cp -R "$example" "$work/$name"
  cd "$work/$name"

  terragrunt hcl format --check
  terraform fmt -check -recursive modules

  for stack in live/*/*/; do
    echo "--> $name/$stack"
    (
      cd "$stack"
      terragrunt stack generate
      terragrunt run --all -- init -backend=false -input=false
      terragrunt run --all --no-auto-init -- validate
    )
  done
done

echo "all examples valid"
