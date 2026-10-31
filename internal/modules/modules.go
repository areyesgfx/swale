// Package modules embeds the vendored Terraform module source that
// internal/generate copies into every generated project's modules/ dir.
// Vendoring means a project never depends on a separate, versioned module
// repo; the trade-off is that swale can't push module fixes into projects
// it has already generated.
package modules

import "embed"

// Networking holds the networking module's source, rooted at "networking"
// (e.g. "networking/main.tf").
//
//go:embed networking
var Networking embed.FS

// Service holds the service module's source, rooted at "service". It runs
// a containerized app on ECS Fargate.
//
//go:embed service
var Service embed.FS

// Database holds the database module's source, rooted at "database". It
// runs PostgreSQL on RDS.
//
//go:embed database
var Database embed.FS
