# 0004: Terragrunt bootstrapped state

## Status

Accepted — 2026-07-01

## Context

Should terragrunt own creating the state bucket? This is a chicken and egg situation where the state backend resources can't be stored in the same state.

## Decision

Yes for simplicity. If at some point we want the state bucket to be configured with settings not available to terragrunt we can change this. Additionally, this avoids using the aws sdk which avoids lots of complexity like managing aws credentials and using the sdk in the application code.

## Consequences

Limited configuration for state bucket. Good for v1.
