package spec

import (
	"fmt"
	"os"

	"go.yaml.in/yaml/v3"
)

// FileName is the name of the spec file at the root of every swale project.
const FileName = "swale.yaml"

// Load reads and decodes a swale.yaml file from path, then validates it.
// Unknown fields are rejected so a typo in a key surfaces as an error
// rather than being silently dropped.
func Load(path string) (Spec, error) {
	f, err := os.Open(path)
	if err != nil {
		return Spec{}, fmt.Errorf("open spec %q: %w", path, err)
	}
	defer f.Close()

	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)

	var s Spec
	if err := dec.Decode(&s); err != nil {
		return Spec{}, fmt.Errorf("decode spec %q: %w", path, err)
	}
	if err := s.Validate(); err != nil {
		return Spec{}, fmt.Errorf("invalid spec %q: %w", path, err)
	}
	return s, nil
}

// Save validates spec and writes it to path as YAML. An invalid spec is
// never written, so any swale.yaml on disk is one swale can load back.
func Save(spec Spec, path string) error {
	if err := spec.Validate(); err != nil {
		return fmt.Errorf("refusing to save invalid spec: %w", err)
	}
	out, err := yaml.Marshal(spec)
	if err != nil {
		return fmt.Errorf("encode spec: %w", err)
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return fmt.Errorf("write spec %q: %w", path, err)
	}
	return nil
}
