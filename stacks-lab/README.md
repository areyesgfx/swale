# stacks-lab

Throwaway prototype for SWA-5.
Not shipped code - it exists to answer questions about Terragrunt stacks
materialization before the real `root.hcl` template (ADR 0005) gets written.
See [FINDINGS.md](FINDINGS.md) for the results.

To reproduce:

```
cd live/dev
terragrunt stack generate
terragrunt stack run apply --non-interactive
```
