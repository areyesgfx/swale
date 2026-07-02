# 0002: Centralized modules repository or generated in project repo

## Status

Accepted — 2026-07-01

## Context

Choosing whether to have a separate dedicated module repository or have a modules/ generated into the project.

## Decision

We will have the modules vendored into the project repo. This keeps things simple for developers as they don't need to manage a separate versioned repo with tags.

## Consequences

Modules are versioned with the project repo. Any fixes made to modules that have already been generated in a project cannot be pushed by swale. Users would have to manually remediate issues.
