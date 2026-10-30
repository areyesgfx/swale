// Package modules embeds the vendored Terraform module source that
// internal/generate copies into every generated project's modules/ dir,
// per ADR 0002 (vendored into the project, no central module repo).
package modules

import "embed"

// Networking holds the networking module's source, rooted at "networking"
// (e.g. "networking/main.tf").
//
//go:embed networking
var Networking embed.FS
