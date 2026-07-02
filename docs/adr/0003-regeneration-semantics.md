# 0003: Regeneration semantics

## Status

Accepted — 2026-07-01

## Context

What happens when someone edits a generated file and re-runs swale?

## Decision

swale will never rewrite an existing file. Add only creates new files, and flags on conflicts. Three-way merging generated code is difficult to manage.

## Consequences

swale can only be used for initial generation of IaC. If users make changes to the IaC, it is no longer owned by Swale.
