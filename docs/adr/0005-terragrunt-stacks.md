# 0005: Terragrunt stacks

## Status

Accepted — 2026-07-01

## Context

Terragrunt stacks are a newer terragrunt feature to simplify IaC even further.

## Decision

Terragrunt stacks will be used. It makes standing up a brand new environment incredibly simple as it is a single logical element. Generated projects should pin the Terragrunt version.

## Consequences

Stacks themselves are a code generator which is what swale is designed to do. So swale using stacks is a generator using a generator. This feature could potentially grow into the direction I am taking swale.

The editable surface becomes the stack files + unit catalogs, while the .terragrunt-stack/ tree is a gitignored build artifact. This conflicts with the "no black box" goal.
