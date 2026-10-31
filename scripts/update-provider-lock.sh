#!/usr/bin/env bash
# Regenerates internal/templates/terraform.lock.hcl, the provider lock file
# swale copies into every generated unit, from the modules' version
# constraints. Run it to upgrade providers, then regenerate examples/ with
# `go test ./internal/generate -update` and review the diff.
#
# Every module must need the same providers, since all units share one
# lock file; the script fails if they don't.
set -euo pipefail

platforms=(darwin_arm64 darwin_amd64 linux_amd64 linux_arm64 windows_amd64)

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

lock_args=()
for p in "${platforms[@]}"; do
  lock_args+=("-platform=$p")
done

first=""
for module in "$repo_root"/internal/modules/*/; do
  name="$(basename "$module")"
  cp -R "$module" "$work/$name"
  terraform -chdir="$work/$name" providers lock "${lock_args[@]}" >/dev/null
  if [ -z "$first" ]; then
    first="$name"
  elif ! diff -q "$work/$first/.terraform.lock.hcl" "$work/$name/.terraform.lock.hcl" >/dev/null; then
    echo "modules $first and $name need different providers; they can't share one lock file" >&2
    exit 1
  fi
done

cp "$work/$first/.terraform.lock.hcl" "$repo_root/internal/templates/terraform.lock.hcl"
echo "updated internal/templates/terraform.lock.hcl"
