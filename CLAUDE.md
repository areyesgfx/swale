# CLAUDE.md

This file provides guidance to Claude Code when working in this repository on this VPS.

## Project

`swale` is a Go CLI (Cobra) — infrastructure/platform tooling, evidenced by the
`iac-mentor-review` skill already in `.claude/skills/` and the Terragrunt stacks templates in
`internal/templates/` (the research behind them is in `docs/research/terragrunt-stacks.md`). Treat this as IaC/platform code: per the owner's standing global instructions, that
means favoring quality, simplicity, robustness, and long-term maintainability over speed of
implementation, being picky about drift and blast radius, and reproducing bugs for real before
proposing fixes rather than reasoning from reading code alone. This is a firmer bar than app
client code (contrast with `st-flutter`, a separate project on this same box, which deliberately
runs looser since it's a solo pre-launch mobile app with no one else depending on `main`).

## Linear

Team: `Swale`, reached via the `linear-swale` MCP tool — **not** `linear`, which points at a
different Linear workspace entirely (the `Soultrip` team, used by the `st-flutter` project on
this same box). Both are registered at Claude Code's `user` scope, so they're available in any
project directory on this VPS, but they are genuinely separate workspaces with no shared teams —
always use `linear-swale` here, double-check you're not accidentally querying `linear`.

## Working the Linear backlog

This repo already has an established ticket-driven history before any of this VPS/T3 Code setup
existed — branches named `areyesgfx/swa-N-description`, opened as PRs, reviewed, and merged
(see `git log --oneline`). Continue that exact pattern, now just executed from this VPS instead
of wherever it ran before:

1. Pick the next unblocked ticket from the `Swale` team in Linear (via `linear-swale`). Move it
   to `In Progress`.
2. `git checkout main && git pull && git checkout -b areyesgfx/swa-<N>-<short-description>` —
   match the existing branch naming convention exactly.
3. Do the work. Reproduce any bug for real before fixing it. Flag drift, unintended blast radius,
   or multi-environment side effects if you notice them, even if unrelated to the immediate
   ticket — per the owner's standing instructions for platform/IaC work.
4. Commit, push the branch, open a PR with `gh pr create` referencing the Linear ticket ID. Move
   the ticket to `In Review`.
5. **Stop there. Wait for the owner to review and merge on GitHub themselves — do not self-merge
   under any circumstances, regardless of confidence or how low-risk the change seems.** This is
   the opposite default from `st-flutter`'s workflow; don't let habits from that project bleed
   into this one. Every PR here waits for explicit human review before merging.
6. Once merged (by the owner, on GitHub): mark the ticket `Done` in Linear, delete the local and
   remote branch, pull the fresh `main`.

## Process notes

- Never add an em dash to anything you write.
- Don't commit or push without the owner explicitly asking, except for the one standing exception
  above: pushing a feature branch and opening its PR is pre-authorized as part of normal ticket
  work. Merging to `main` is never pre-authorized here.
- No `Co-Authored-By` line in commit messages.
- For technical decisions, don't weigh development speed heavily — prefer quality, simplicity,
  robustness, and scalability, especially since this is platform/IaC code others may build on.
