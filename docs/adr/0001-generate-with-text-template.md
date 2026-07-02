# 0001: Generate IaC with text/template, not hclwrite

## Status

Accepted — 2026-07-01

## Context

Choice of which approach for writing the IaC. Options were text/template or hclwrite. hclwrite would be a better choice if we wanted swale to edit user-modified files.

## Decision

We will use text/template because we don't want swale to edit user-modified files which is out of scope for v1. Additionally, text/template also provides a more readable/editable output. 

## Consequences

swale can render HCL but cannot parse or inspect it. This means any future features like the ability to edit existing files will require changing away from text/template.
