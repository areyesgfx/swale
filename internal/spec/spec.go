// Package spec defines Spec, the contents of swale.yaml: what an app needs,
// in the terms of the developer building it. Both front doors (the
// interactive wizard and CLI flags) produce a Spec, and internal/generate
// derives the infrastructure from it, so nothing in a Spec requires knowing
// AWS networking.
package spec

import (
	"encoding/hex"
	"fmt"
	"io"
)

// Spec is the root of a swale.yaml file. Environments are a subset of
// KnownEnvironments, in any order.
//
// StateBucket is picked once, when the project is created, rather than
// derived from Name: S3 bucket names are global across every AWS account,
// so a name-derived bucket would collide with any other swale project
// that has the same name.
type Spec struct {
	Name         string   `yaml:"name"`
	Region       string   `yaml:"region"`
	Environments []string `yaml:"environments"`
	StateBucket  string   `yaml:"state_bucket"`
}

// KnownEnvironments lists every environment swale knows how to build,
// from least to most critical.
var KnownEnvironments = []string{"dev", "stage", "prod"}

// EnvironmentsFor answers "how many environments do you need?". Smaller
// projects drop the least critical environments first, so prod is always
// there and the first environment in the list is the safest place to try
// a deploy.
func EnvironmentsFor(count int) ([]string, error) {
	if count < 1 || count > len(KnownEnvironments) {
		return nil, fmt.Errorf("swale supports 1 to %d environments, got %d", len(KnownEnvironments), count)
	}
	envs := KnownEnvironments[len(KnownEnvironments)-count:]
	return append([]string(nil), envs...), nil
}

// NewStateBucket returns a state bucket name for an app called name, with
// a random suffix read from random (crypto/rand.Reader outside tests) so
// it doesn't collide with another project's bucket.
func NewStateBucket(name string, random io.Reader) (string, error) {
	suffix := make([]byte, 4)
	if _, err := io.ReadFull(random, suffix); err != nil {
		return "", fmt.Errorf("pick a state bucket name: %w", err)
	}
	return fmt.Sprintf("%s-tfstate-%s", name, hex.EncodeToString(suffix)), nil
}
